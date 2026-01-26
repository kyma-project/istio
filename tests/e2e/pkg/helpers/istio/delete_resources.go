package istiohelpers

import (
	"testing"

	"github.com/stretchr/testify/require"
	v3 "istio.io/client-go/pkg/apis/security/v1"
	v2 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

// DeleteIstiod deletes the istiod deployment from istio-system namespace
func DeleteIstiod(t *testing.T, c *resources.Resources) {
	t.Helper()
	err := c.Delete(t.Context(), &v2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istiod",
			Namespace: "istio-system",
		},
	})
	require.NoError(t, err)
}

// DeleteIngressGateway deletes the istio-ingressgateway deployment from istio-system namespace
func DeleteIngressGateway(t *testing.T, c *resources.Resources) {
	t.Helper()
	err := c.Delete(t.Context(), &v2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-ingressgateway",
			Namespace: "istio-system",
		},
	})
	require.NoError(t, err)
}

// DeleteEgressGateway deletes the istio-egressgateway deployment from istio-system namespace
func DeleteEgressGateway(t *testing.T, c *resources.Resources) {
	t.Helper()
	err := c.Delete(t.Context(), &v2.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-egressgateway",
			Namespace: "istio-system",
		},
	})
	require.NoError(t, err)
}

// DeleteCNINode deletes the istio-cni-node daemonset from istio-system namespace
func DeleteCNINode(t *testing.T, c *resources.Resources) {
	t.Helper()
	err := c.Delete(t.Context(), &v2.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-cni-node",
			Namespace: "istio-system",
		},
	})
	require.NoError(t, err)
}

// DeleteDefaultPeerAuthentication deletes the default PeerAuthentication from istio-system namespace
func DeleteDefaultPeerAuthentication(t *testing.T, c *resources.Resources) {
	t.Helper()
	err := c.Delete(t.Context(), &v3.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "istio-system",
		},
	})
	require.NoError(t, err)
}
