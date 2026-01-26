package istioassert

import (
	"testing"

	"github.com/stretchr/testify/require"
	v3 "istio.io/client-go/pkg/apis/security/v1"
	v2 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	resourceassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/resources"
)

// AssertIstiodReady gets the istiod deployment and asserts it is available
func AssertIstiodReady(t *testing.T, c *resources.Resources) {
	t.Helper()

	istiodDeployment := &v2.Deployment{}
	err := c.Get(t.Context(), "istiod", "istio-system", istiodDeployment)
	require.NoError(t, err)
	err = wait.For(conditions.New(c).DeploymentConditionMatch(istiodDeployment, v2.DeploymentAvailable, v1.ConditionTrue), wait.WithContext(t.Context()))
	require.NoError(t, err)
}

// AssertIngressGatewayReady gets the istio-ingressgateway deployment and asserts it is available
func AssertIngressGatewayReady(t *testing.T, c *resources.Resources) {
	t.Helper()

	ingressDeployment := &v2.Deployment{}
	err := c.Get(t.Context(), "istio-ingressgateway", "istio-system", ingressDeployment)
	require.NoError(t, err)
	err = wait.For(conditions.New(c).DeploymentConditionMatch(ingressDeployment, v2.DeploymentAvailable, v1.ConditionTrue), wait.WithContext(t.Context()))
	require.NoError(t, err)
}

// AssertEgressGatewayReady gets the istio-egressgateway deployment and asserts it is available
func AssertEgressGatewayReady(t *testing.T, c *resources.Resources) {
	t.Helper()

	egressDeployment := &v2.Deployment{}
	err := c.Get(t.Context(), "istio-egressgateway", "istio-system", egressDeployment)
	require.NoError(t, err)
	err = wait.For(conditions.New(c).DeploymentConditionMatch(egressDeployment, v2.DeploymentAvailable, v1.ConditionTrue), wait.WithContext(t.Context()))
	require.NoError(t, err)
}

// AssertCNINodeReady gets the istio-cni-node daemonset and asserts it is ready
func AssertCNINodeReady(t *testing.T, c *resources.Resources) {
	t.Helper()

	cniDaemonSet := &v2.DaemonSet{}
	err := c.Get(t.Context(), "istio-cni-node", "istio-system", cniDaemonSet)
	require.NoError(t, err)
	err = wait.For(conditions.New(c).DaemonSetReady(cniDaemonSet), wait.WithContext(t.Context()))
	require.NoError(t, err)
}

// AssertIstiodPodResources asserts that all istiod pods have the expected resource requests and limits
func AssertIstiodPodResources(t *testing.T, c *resources.Resources, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory string) {
	t.Helper()

	istiodPodList := &v1.PodList{}
	err := c.List(t.Context(), istiodPodList, resources.WithLabelSelector("app=istiod"))
	require.NoError(t, err)

	for _, pod := range istiodPodList.Items {
		istiod := pod.Spec.Containers[0]
		require.Equal(t, "discovery", istiod.Name)
		resourceassert.AssertContainerResources(t, istiod, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory)
	}
}

// AssertIngressGatewayPodResources asserts that all istio-ingressgateway pods have the expected resource requests and limits
func AssertIngressGatewayPodResources(t *testing.T, c *resources.Resources, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory string) {
	t.Helper()

	ingressPodList := &v1.PodList{}
	err := c.List(t.Context(), ingressPodList, resources.WithLabelSelector("app=istio-ingressgateway"))
	require.NoError(t, err)

	for _, pod := range ingressPodList.Items {
		ingress := pod.Spec.Containers[0]
		require.Equal(t, "istio-proxy", ingress.Name)
		resourceassert.AssertContainerResources(t, ingress, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory)
	}
}

// AssertEgressGatewayPodResources asserts that all istio-egressgateway pods have the expected resource requests and limits
func AssertEgressGatewayPodResources(t *testing.T, c *resources.Resources, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory string) {
	t.Helper()

	egressPodList := &v1.PodList{}
	err := c.List(t.Context(), egressPodList, resources.WithLabelSelector("app=istio-egressgateway"))
	require.NoError(t, err)

	for _, pod := range egressPodList.Items {
		egress := pod.Spec.Containers[0]
		require.Equal(t, "istio-proxy", egress.Name)
		resourceassert.AssertContainerResources(t, egress, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory)
	}
}

// AssertDefaultPeerAuthenticationExists asserts that the default PeerAuthentication exists in istio-system namespace
func AssertDefaultPeerAuthenticationExists(t *testing.T, c *resources.Resources) *v3.PeerAuthentication {
	t.Helper()

	pa := &v3.PeerAuthentication{}
	err := c.Get(t.Context(), "default", "istio-system", pa)
	require.NoError(t, err)
	return pa
}
