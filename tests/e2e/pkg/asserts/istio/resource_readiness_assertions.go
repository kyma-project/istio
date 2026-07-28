package istioassert

import (
	"testing"

	"github.com/stretchr/testify/require"
	istiosecurityv1 "istio.io/client-go/pkg/apis/security/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	resourceassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/resources"
)

// AssertIstiodReady gets the istiod deployment and asserts it is available
func AssertIstiodReady(t *testing.T, c *resources.Resources) {
	t.Helper()
	require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable("istiod", "istio-system"), wait.WithContext(t.Context())))
}

// AssertIngressGatewayReady gets the istio-ingressgateway deployment and asserts it is available
func AssertIngressGatewayReady(t *testing.T, c *resources.Resources) {
	t.Helper()
	require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable("istio-ingressgateway", "istio-system"), wait.WithContext(t.Context())))
}

// AssertEgressGatewayReady gets the istio-egressgateway deployment and asserts it is available
func AssertEgressGatewayReady(t *testing.T, c *resources.Resources) {
	t.Helper()
	require.NoError(t, wait.For(conditions.New(c).DeploymentAvailable("istio-egressgateway", "istio-system"), wait.WithContext(t.Context())))
}

// AssertCNINodeReady gets the istio-cni-node daemonset and asserts it is ready
func AssertCNINodeReady(t *testing.T, c *resources.Resources) {
	t.Helper()
	ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "istio-cni-node", Namespace: "istio-system"}}
	require.NoError(t, wait.For(conditions.New(c).DaemonSetReady(&ds), wait.WithContext(t.Context())))
}

// AssertZtunnelReady gets the ztunnel daemonset and asserts it is ready
func AssertZtunnelReady(t *testing.T, c *resources.Resources) {
	t.Helper()
	ds := appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "ztunnel", Namespace: "istio-system"}}
	require.NoError(t, wait.For(conditions.New(c).DaemonSetReady(&ds), wait.WithContext(t.Context())))
}

// AssertIstiodPodResources asserts that all istiod pods have the expected resource requests and limits
func AssertIstiodPodResources(t *testing.T, c *resources.Resources, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory string) {
	t.Helper()

	istiodPodList := &corev1.PodList{}
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

	ingressPodList := &corev1.PodList{}
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

	egressPodList := &corev1.PodList{}
	err := c.List(t.Context(), egressPodList, resources.WithLabelSelector("app=istio-egressgateway"))
	require.NoError(t, err)

	for _, pod := range egressPodList.Items {
		egress := pod.Spec.Containers[0]
		require.Equal(t, "istio-proxy", egress.Name)
		resourceassert.AssertContainerResources(t, egress, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory)
	}
}

// AssertDefaultPeerAuthenticationExists asserts that the default PeerAuthentication exists in istio-system namespace
func AssertDefaultPeerAuthenticationExists(t *testing.T, c *resources.Resources) *istiosecurityv1.PeerAuthentication {
	t.Helper()

	pa := &istiosecurityv1.PeerAuthentication{}
	err := c.Get(t.Context(), "default", "istio-system", pa)
	require.NoError(t, err)
	return pa
}

// AssertPodDisruptionBudgetMinAvailable asserts that the PodDisruptionBudget exists in the given namespace and has the expected minAvailable value
func AssertPodDisruptionBudgetMinAvailable(t *testing.T, c *resources.Resources, name, namespace string, expectedMinAvailable int32) {
	t.Helper()

	pdb := &policyv1.PodDisruptionBudget{}
	err := c.Get(t.Context(), name, namespace, pdb)
	require.NoError(t, err)

	require.NotNil(t, pdb.Spec.MinAvailable, "PDB %s/%s has no MinAvailable", namespace, name)
	require.Equal(t, expectedMinAvailable, pdb.Spec.MinAvailable.IntVal, "PDB %s/%s MinAvailable mismatch", namespace, name)
	require.NotNil(t, pdb.Spec.Selector, "PDB %s/%s has no selector", namespace, name)
}

// AssertPodDisruptionBudgetMaxUnavailable asserts that the PodDisruptionBudget exists in the given namespace and has the expected maxUnavailable value
func AssertPodDisruptionBudgetMaxUnavailable(t *testing.T, c *resources.Resources, name, namespace string, expectedMaxUnavailable int32) {
	t.Helper()

	pdb := &policyv1.PodDisruptionBudget{}
	err := c.Get(t.Context(), name, namespace, pdb)
	require.NoError(t, err)

	require.NotNil(t, pdb.Spec.MaxUnavailable, "PDB %s/%s has no MaxUnavailable", namespace, name)
	require.Equal(t, expectedMaxUnavailable, pdb.Spec.MaxUnavailable.IntVal, "PDB %s/%s MaxUnavailable mismatch", namespace, name)
	require.NotNil(t, pdb.Spec.Selector, "PDB %s/%s has no selector", namespace, name)
}
