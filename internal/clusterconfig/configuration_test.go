package clusterconfig_test

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EvaluateClusterConfiguration Tests

func TestEvaluateClusterConfiguration_K3d(t *testing.T) {
	// Given
	node := newK3dNode("k3d-node-1")
	testClient := newTestClient(t, node)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	expected := clusterconfig.ClusterConfiguration(map[string]interface{}{
		"spec": map[string]interface{}{
			"values": map[string]interface{}{
				"cni": map[string]string{
					"cniBinDir":  "/var/lib/rancher/k3s/data/cni",
					"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
				},
			},
		},
	})
	assert.Equal(t, expected, config)
}

func TestEvaluateClusterConfiguration_GKE(t *testing.T) {
	// Given
	node := newGKENode("gke-node-1")
	testClient := newTestClient(t, node)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	expected := clusterconfig.ClusterConfiguration(map[string]interface{}{
		"spec": map[string]interface{}{
			"values": map[string]interface{}{
				"cni": map[string]interface{}{
					"cniBinDir": "/home/kubernetes/bin",
					"resourceQuotas": map[string]bool{
						"enabled": true,
					},
				},
			},
		},
	})
	assert.Equal(t, expected, config)
}

func TestEvaluateClusterConfiguration_AWS_NLB(t *testing.T) {
	// Given
	node := newAWSNode("aws-node-1")
	testClient := newTestClient(t, node)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.AWSNLBConfig, config)
}

func TestEvaluateClusterConfiguration_AWS_NLB_WithELBDeprecatedAndNLBService(t *testing.T) {
	// Given
	node := newAWSNode("aws-node-1")
	elbDeprecatedCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elb-deprecated",
			Namespace: "istio-system",
		},
	}
	ingressService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-ingressgateway",
			Namespace: "istio-system",
			Annotations: map[string]string{
				"service.beta.kubernetes.io/aws-load-balancer-scheme":          "internet-facing",
				"service.beta.kubernetes.io/aws-load-balancer-nlb-target-type": "instance",
				"service.beta.kubernetes.io/aws-load-balancer-type":            "nlb",
			},
		},
	}
	testClient := newTestClient(t, node, elbDeprecatedCM, ingressService)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.AWSNLBConfig, config)
}

func TestEvaluateClusterConfiguration_AWS_LegacyELB(t *testing.T) {
	// Given
	node := newAWSNode("aws-node-1")
	elbDeprecatedCM := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "elb-deprecated",
			Namespace: "istio-system",
		},
	}
	testClient := newTestClient(t, node, elbDeprecatedCM)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.ClusterConfiguration{}, config)
}

func TestEvaluateClusterConfiguration_GardenerOpenStack(t *testing.T) {
	// Given
	node := newGardenerNode("gardener-node-1", "openstack://example")
	testClient := newTestClient(t, node)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.OpenStackLBProxyProtocolConfig, config)
}

func TestEvaluateClusterConfiguration_GardenerAWS(t *testing.T) {
	// Given
	node := newGardenerNode("gardener-node-1", "aws://example")
	testClient := newTestClient(t, node)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.AWSNLBConfig, config)
}

func TestEvaluateClusterConfiguration_UnknownCluster(t *testing.T) {
	// Given
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "unknown-node-1",
		},
	}
	testClient := newTestClient(t, node)
	provider, err := clusterconfig.GetClusterProvider(context.Background(), testClient)
	require.NoError(t, err)

	// When
	config, err := clusterconfig.EvaluateClusterConfiguration(context.Background(), testClient, provider)

	// Then
	require.NoError(t, err)
	assert.Equal(t, clusterconfig.ClusterConfiguration{}, config)
}
