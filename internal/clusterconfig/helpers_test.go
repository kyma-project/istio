package clusterconfig_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
)

// Test client helpers

func newTestClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, networkingv1alpha3.AddToScheme(s))

	return fake.NewClientBuilder().
		WithScheme(s).
		WithObjects(objects...).
		Build()
}

// Node factory helpers

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

func newGardenerNode(name, provider string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: corev1.NodeSpec{
			ProviderID: provider,
		},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				OSImage: "Garden Linux 12.04",
			},
		},
	}
}

// Configuration extraction helpers

func extractServiceAnnotations(config clusterconfig.ClusterConfiguration) map[string]string {
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
