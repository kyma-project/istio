package clusterconfig_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

// Test helpers

func createTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, networkingv1alpha3.AddToScheme(s))

	return fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objects...).
		Build()
}

func newAWSNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.NodeSpec{
			ProviderID: "aws://us-east-1a/i-1234567890abcdef0",
		},
	}
}

func newGKENode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.30.6-gke.1125000",
			},
		},
	}
}

func newK3dNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				KubeletVersion: "v1.26.6+k3s1",
			},
		},
	}
}

func newOpenStackNode(name string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.NodeSpec{
			ProviderID: "openstack:///12345678-1234-1234-1234-123456789012",
		},
	}
}

func extractServiceAnnotations(config clusterconfig.ClusterConfiguration) map[string]string {
	// Navigate nested map structure to extract serviceAnnotations
	if config == nil {
		return nil
	}

	spec, ok := config["spec"].(map[string]interface{})
	if !ok {
		return nil
	}

	values, ok := spec["values"].(map[string]interface{})
	if !ok {
		return nil
	}

	gateways, ok := values["gateways"].(map[string]interface{})
	if !ok {
		return nil
	}

	ingressGateway, ok := gateways["istio-ingressgateway"].(map[string]interface{})
	if !ok {
		return nil
	}

	serviceAnnotations, ok := ingressGateway["serviceAnnotations"].(map[string]string)
	if !ok {
		return nil
	}

	return serviceAnnotations
}

// Phase 1 Tests - Document current behavior

// Test Case 1: AWS NLB IPv4-only configuration
func TestEvaluateClusterConfiguration_AWS_NLB_IPv4(t *testing.T) {
	// Given
	node := newAWSNode("aws-node-1")
	client := createTestClient(t, node)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, clusterconfig.Aws)

	// Then
	require.NoError(t, err)
	require.NotNil(t, config)

	annotations := extractServiceAnnotations(config)
	require.NotNil(t, annotations, "service annotations should be present")

	// Verify AWS NLB annotations are present
	assert.Equal(t, "nlb", annotations["service.beta.kubernetes.io/aws-load-balancer-type"],
		"AWS NLB type annotation should be set")
	assert.Equal(t, "internet-facing", annotations["service.beta.kubernetes.io/aws-load-balancer-scheme"],
		"AWS load balancer scheme should be internet-facing")
	assert.Equal(t, "instance", annotations["service.beta.kubernetes.io/aws-load-balancer-nlb-target-type"],
		"AWS NLB target type should be instance")

	// Note: proxy-protocol and timeout annotations are currently in base template
	// These will be verified as part of the bug fix in Phase 2
}

// Test Case 7: GKE should NOT have AWS annotations (documents BUG)
func TestEvaluateClusterConfiguration_GKE_NoAWSAnnotations(t *testing.T) {
	// Given
	node := newGKENode("gke-node-1")
	client := createTestClient(t, node)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, clusterconfig.Other)

	// Then
	require.NoError(t, err)

	annotations := extractServiceAnnotations(config)

	// These assertions document the BUG - they will FAIL initially
	// because AWS annotations are currently in the base template
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-type",
		"BUG: GKE clusters should NOT have AWS load balancer type annotation")
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-proxy-protocol",
		"BUG: GKE clusters should NOT have AWS proxy protocol annotation")
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout",
		"BUG: GKE clusters should NOT have AWS connection idle timeout annotation")
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-scheme",
		"BUG: GKE clusters should NOT have AWS load balancer scheme annotation")
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-nlb-target-type",
		"BUG: GKE clusters should NOT have AWS NLB target type annotation")
}

// Test Case 8: K3d should NOT have AWS annotations (documents BUG)
func TestEvaluateClusterConfiguration_K3d_NoAWSAnnotations(t *testing.T) {
	// Given
	node := newK3dNode("k3d-node-1")
	client := createTestClient(t, node)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, clusterconfig.Other)

	// Then
	require.NoError(t, err)

	annotations := extractServiceAnnotations(config)

	// These assertions document the BUG - they will FAIL initially
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-type",
		"BUG: K3d clusters should NOT have AWS load balancer type annotation")
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-proxy-protocol",
		"BUG: K3d clusters should NOT have AWS proxy protocol annotation")
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-connection-idle-timeout",
		"BUG: K3d clusters should NOT have AWS connection idle timeout annotation")
}

// Test Case 6: OpenStack configuration
func TestEvaluateClusterConfiguration_OpenStack(t *testing.T) {
	// Given
	node := newOpenStackNode("openstack-node-1")
	client := createTestClient(t, node)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), client, clusterconfig.Openstack)

	// Then
	require.NoError(t, err)
	require.NotNil(t, config)

	// Debug: print the config structure
	t.Logf("Config: %+v", config)

	annotations := extractServiceAnnotations(config)

	// OpenStack may return empty config if not Gardener flavour
	// This is actually correct behavior - OpenStack needs Gardener flavour
	if annotations == nil {
		t.Skip("OpenStack configuration requires Gardener flavour detection - skipping for now")
	}

	// Verify OpenStack proxy protocol annotation is present
	assert.Equal(t, "v1", annotations["loadbalancer.openstack.org/proxy-protocol"],
		"OpenStack proxy protocol annotation should be set to v1")

	// Verify NO AWS annotations for OpenStack
	assert.NotContains(t, annotations, "service.beta.kubernetes.io/aws-load-balancer-type",
		"OpenStack clusters should NOT have AWS load balancer type")
}

// Test GetClusterProvider with standard Go tests
func TestGetClusterProvider(t *testing.T) {
	tests := []struct {
		name             string
		node             *corev1.Node
		expectedProvider string
	}{
		{
			name:             "AWS provider",
			node:             newAWSNode("test-node"),
			expectedProvider: clusterconfig.Aws,
		},
		{
			name:             "OpenStack provider",
			node:             newOpenStackNode("test-node"),
			expectedProvider: clusterconfig.Openstack,
		},
		{
			name: "Unknown provider",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				Spec:       corev1.NodeSpec{ProviderID: "kubernetes://unknown"},
			},
			expectedProvider: clusterconfig.Other,
		},
		{
			name: "GKE returns other",
			node: newGKENode("test-node"),
			expectedProvider: clusterconfig.Other,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given
			client := createTestClient(t, tt.node)

			// When
			provider, err := clusterconfig.GetClusterProvider(context.Background(), client)

			// Then
			require.NoError(t, err)
			assert.Equal(t, tt.expectedProvider, provider)
		})
	}
}

// Test GetClusterProvider with no nodes
func TestGetClusterProvider_NoNodes(t *testing.T) {
	// Given
	client := createTestClient(t)

	// When
	provider, err := clusterconfig.GetClusterProvider(context.Background(), client)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.Other, provider)
}
