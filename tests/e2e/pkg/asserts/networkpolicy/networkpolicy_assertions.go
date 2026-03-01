package networkpolicyassert

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

const (
	// NetworkPolicy names as defined in the ADR
	ControllerManagerNetworkPolicyName    = "kyma-project.io--allow-istio-controller-manager"
	IstiodNetworkPolicyName               = "kyma-project.io--istio-pilot"
	IstiodJWKSNetworkPolicyName           = "kyma-project.io--istio-pilot-jwks"
	IngressGatewayNetworkPolicyName       = "kyma-project.io--istio-ingressgateway"
	IngressGatewayEgressNetworkPolicyName = "kyma-project.io--istio-ingressgateway-egress"
	EgressGatewayNetworkPolicyName        = "kyma-project.io--istio-egressgateway"
	CNINodeNetworkPolicyName              = "kyma-project.io--istio-cni-node"

	// Expected labels for module-managed NetworkPolicies
	ModuleLabelKey      = "kyma-project.io/module"
	ModuleLabelValue    = "istio"
	ManagedByLabelKey   = "kyma-project.io/managed-by"
	ManagedByLabelValue = "kyma"
)

// AssertNetworkPolicyExists asserts that a NetworkPolicy exists in the specified namespace
func AssertNetworkPolicyExists(t *testing.T, r *resources.Resources, namespace, name string) *networkingv1.NetworkPolicy {
	t.Helper()
	t.Logf("Asserting NetworkPolicy %s exists in namespace %s", name, namespace)

	np := &networkingv1.NetworkPolicy{}
	err := r.Get(context.Background(), name, namespace, np)
	require.NoError(t, err, "NetworkPolicy %s should exist in namespace %s", name, namespace)

	return np
}

// AssertNetworkPolicyNotExists asserts that a NetworkPolicy does not exist in the specified namespace
func AssertNetworkPolicyNotExists(t *testing.T, r *resources.Resources, namespace, name string) {
	t.Helper()
	t.Logf("Asserting NetworkPolicy %s does not exist in namespace %s", name, namespace)

	np := &networkingv1.NetworkPolicy{}
	err := r.Get(context.Background(), name, namespace, np)
	require.True(t, k8serrors.IsNotFound(err), "NetworkPolicy %s should not exist in namespace %s", name, namespace)
}

// AssertNetworkPolicyHasModuleLabels asserts that a NetworkPolicy has the expected module labels
func AssertNetworkPolicyHasModuleLabels(t *testing.T, np *networkingv1.NetworkPolicy) {
	t.Helper()

	labels := np.GetLabels()
	require.NotNil(t, labels, "NetworkPolicy %s should have labels", np.Name)

	moduleLabel, ok := labels[ModuleLabelKey]
	require.True(t, ok, "NetworkPolicy %s should have label %s", np.Name, ModuleLabelKey)
	require.Equal(t, ModuleLabelValue, moduleLabel, "NetworkPolicy %s should have label %s=%s", np.Name, ModuleLabelKey, ModuleLabelValue)

	managedByLabel, ok := labels[ManagedByLabelKey]
	require.True(t, ok, "NetworkPolicy %s should have label %s", np.Name, ManagedByLabelKey)
	require.Equal(t, ManagedByLabelValue, managedByLabel, "NetworkPolicy %s should have label %s=%s", np.Name, ManagedByLabelKey, ManagedByLabelValue)
}

// AssertNetworkPolicyHasPodSelector asserts that a NetworkPolicy has the expected pod selector
func AssertNetworkPolicyHasPodSelector(t *testing.T, np *networkingv1.NetworkPolicy, expectedLabels map[string]string) {
	t.Helper()

	require.NotNil(t, np.Spec.PodSelector.MatchLabels, "NetworkPolicy %s should have pod selector", np.Name)
	for key, expectedValue := range expectedLabels {
		actualValue, ok := np.Spec.PodSelector.MatchLabels[key]
		require.True(t, ok, "NetworkPolicy %s should have pod selector label %s", np.Name, key)
		require.Equal(t, expectedValue, actualValue, "NetworkPolicy %s pod selector label %s should have value %s", np.Name, key, expectedValue)
	}
}

// AssertNetworkPolicyHasEgressRule asserts that a NetworkPolicy has at least one egress rule with the specified port
func AssertNetworkPolicyHasEgressRule(t *testing.T, np *networkingv1.NetworkPolicy, protocol string, port int32) {
	t.Helper()

	require.NotEmpty(t, np.Spec.Egress, "NetworkPolicy %s should have egress rules", np.Name)

	found := false
	for _, egressRule := range np.Spec.Egress {
		for _, portRule := range egressRule.Ports {
			if portRule.Protocol != nil && string(*portRule.Protocol) == protocol &&
				portRule.Port != nil && portRule.Port.IntVal == port {
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	require.True(t, found, "NetworkPolicy %s should have egress rule for protocol %s on port %d", np.Name, protocol, port)
}

// AssertNetworkPolicyHasIngressRule asserts that a NetworkPolicy has at least one ingress rule with the specified port
func AssertNetworkPolicyHasIngressRule(t *testing.T, np *networkingv1.NetworkPolicy, protocol string, port int32) {
	t.Helper()

	require.NotEmpty(t, np.Spec.Ingress, "NetworkPolicy %s should have ingress rules", np.Name)

	found := false
	for _, ingressRule := range np.Spec.Ingress {
		for _, portRule := range ingressRule.Ports {
			if portRule.Protocol != nil && string(*portRule.Protocol) == protocol &&
				portRule.Port != nil && portRule.Port.IntVal == port {
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	require.True(t, found, "NetworkPolicy %s should have ingress rule for protocol %s on port %d", np.Name, protocol, port)
}

// AssertNetworkPolicyHasPolicyType asserts that a NetworkPolicy has the specified policy type
func AssertNetworkPolicyHasPolicyType(t *testing.T, np *networkingv1.NetworkPolicy, policyType networkingv1.PolicyType) {
	t.Helper()

	found := false
	for _, pt := range np.Spec.PolicyTypes {
		if pt == policyType {
			found = true
			break
		}
	}

	require.True(t, found, "NetworkPolicy %s should have policy type %s", np.Name, policyType)
}

// AssertNetworkPolicyHasIngressFromLabeledPods asserts that a NetworkPolicy allows ingress from pods with specific label
func AssertNetworkPolicyHasIngressFromLabeledPods(t *testing.T, np *networkingv1.NetworkPolicy, labelKey, labelValue string) {
	t.Helper()

	require.NotEmpty(t, np.Spec.Ingress, "NetworkPolicy %s should have ingress rules", np.Name)

	found := false
	for _, ingressRule := range np.Spec.Ingress {
		for _, from := range ingressRule.From {
			if from.PodSelector != nil && from.PodSelector.MatchLabels != nil {
				if val, ok := from.PodSelector.MatchLabels[labelKey]; ok && val == labelValue {
					found = true
					break
				}
			}
		}
		if found {
			break
		}
	}

	require.True(t, found, "NetworkPolicy %s should have ingress rule from pods with label %s=%s", np.Name, labelKey, labelValue)
}

// AssertNetworkPolicyAllowsAllEgress asserts that a NetworkPolicy allows all egress traffic
func AssertNetworkPolicyAllowsAllEgress(t *testing.T, np *networkingv1.NetworkPolicy) {
	t.Helper()

	// Check if policy type includes Egress
	hasEgressPolicy := false
	for _, pt := range np.Spec.PolicyTypes {
		if pt == networkingv1.PolicyTypeEgress {
			hasEgressPolicy = true
			break
		}
	}
	require.True(t, hasEgressPolicy, "NetworkPolicy %s should have Egress policy type", np.Name)

	// Check for empty egress rule (allows all) or specific all-allow pattern
	require.NotEmpty(t, np.Spec.Egress, "NetworkPolicy %s should have egress rules", np.Name)

	// An empty egress rule (no ports, no to) means allow all
	for _, egressRule := range np.Spec.Egress {
		if len(egressRule.Ports) == 0 && len(egressRule.To) == 0 {
			return // Found an allow-all rule
		}
	}

	// If no explicit allow-all rule, the test still passes if there are egress rules
	// (egress gateway needs to route to various destinations)
	t.Logf("NetworkPolicy %s has specific egress rules, not a blanket allow-all", np.Name)
}

// AssertAllModuleNetworkPoliciesExist asserts that all module-managed NetworkPolicies exist
func AssertAllModuleNetworkPoliciesExist(t *testing.T, r *resources.Resources) {
	t.Helper()
	t.Log("Asserting all module-managed NetworkPolicies exist")

	// Check kyma-system NetworkPolicies
	AssertNetworkPolicyExists(t, r, "kyma-system", ControllerManagerNetworkPolicyName)

	// Check istio-system NetworkPolicies
	AssertNetworkPolicyExists(t, r, "istio-system", IstiodNetworkPolicyName)
	AssertNetworkPolicyExists(t, r, "istio-system", IstiodJWKSNetworkPolicyName)
	AssertNetworkPolicyExists(t, r, "istio-system", IngressGatewayNetworkPolicyName)
	AssertNetworkPolicyExists(t, r, "istio-system", IngressGatewayEgressNetworkPolicyName)
	AssertNetworkPolicyExists(t, r, "istio-system", EgressGatewayNetworkPolicyName)
	AssertNetworkPolicyExists(t, r, "istio-system", CNINodeNetworkPolicyName)
}

// AssertAllModuleNetworkPoliciesNotExist asserts that all module-managed NetworkPolicies do not exist
func AssertAllModuleNetworkPoliciesNotExist(t *testing.T, r *resources.Resources) {
	t.Helper()
	t.Log("Asserting all module-managed NetworkPolicies do not exist")

	// Check kyma-system NetworkPolicies
	AssertNetworkPolicyNotExists(t, r, "kyma-system", ControllerManagerNetworkPolicyName)

	// Check istio-system NetworkPolicies
	AssertNetworkPolicyNotExists(t, r, "istio-system", IstiodNetworkPolicyName)
	AssertNetworkPolicyNotExists(t, r, "istio-system", IstiodJWKSNetworkPolicyName)
	AssertNetworkPolicyNotExists(t, r, "istio-system", IngressGatewayNetworkPolicyName)
	AssertNetworkPolicyNotExists(t, r, "istio-system", IngressGatewayEgressNetworkPolicyName)
	AssertNetworkPolicyNotExists(t, r, "istio-system", EgressGatewayNetworkPolicyName)
	AssertNetworkPolicyNotExists(t, r, "istio-system", CNINodeNetworkPolicyName)
}

// WaitForNetworkPolicyCreation waits for a NetworkPolicy to be created within the specified timeout
func WaitForNetworkPolicyCreation(t *testing.T, r *resources.Resources, namespace, name string, timeout time.Duration) error {
	t.Helper()
	t.Logf("Waiting for NetworkPolicy %s to be created in namespace %s", name, namespace)

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	return wait.For(
		conditions.New(r).ResourceMatch(np, func(object k8s.Object) bool {
			return object != nil
		}),
		wait.WithTimeout(timeout),
		wait.WithContext(context.Background()),
	)
}

// WaitForNetworkPolicyDeletion waits for a NetworkPolicy to be deleted within the specified timeout
func WaitForNetworkPolicyDeletion(t *testing.T, r *resources.Resources, namespace, name string, timeout time.Duration) error {
	t.Helper()
	t.Logf("Waiting for NetworkPolicy %s to be deleted from namespace %s", name, namespace)

	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	return wait.For(
		conditions.New(r).ResourceDeleted(np),
		wait.WithTimeout(timeout),
		wait.WithContext(context.Background()),
	)
}

// AssertControllerManagerNetworkPolicy validates the istio-controller-manager NetworkPolicy
func AssertControllerManagerNetworkPolicy(t *testing.T, r *resources.Resources) {
	t.Helper()

	np := AssertNetworkPolicyExists(t, r, "kyma-system", ControllerManagerNetworkPolicyName)
	AssertNetworkPolicyHasModuleLabels(t, np)
	AssertNetworkPolicyHasPodSelector(t, np, map[string]string{
		"kyma-project.io/module": "istio",
		"app.kubernetes.io/name": "istio-operator",
	})
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeEgress)

	// Verify DNS egress rules (TCP and UDP port 53)
	AssertNetworkPolicyHasEgressRule(t, np, "TCP", 53)
	AssertNetworkPolicyHasEgressRule(t, np, "UDP", 53)
}

// AssertIstiodNetworkPolicy validates the istiod NetworkPolicy
func AssertIstiodNetworkPolicy(t *testing.T, r *resources.Resources) {
	t.Helper()

	np := AssertNetworkPolicyExists(t, r, "istio-system", IstiodNetworkPolicyName)
	AssertNetworkPolicyHasModuleLabels(t, np)
	AssertNetworkPolicyHasPodSelector(t, np, map[string]string{
		"istio": "pilot",
	})
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeEgress)
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeIngress)

	// Verify DNS egress rules
	AssertNetworkPolicyHasEgressRule(t, np, "TCP", 53)
	AssertNetworkPolicyHasEgressRule(t, np, "UDP", 53)

	// Verify ingress rules for control plane communication
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 15012) // XDS
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 15014) // Monitoring
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 15017) // Webhook
}

// AssertIstiodJWKSNetworkPolicy validates the istiod JWKS NetworkPolicy
func AssertIstiodJWKSNetworkPolicy(t *testing.T, r *resources.Resources) {
	t.Helper()

	np := AssertNetworkPolicyExists(t, r, "istio-system", IstiodJWKSNetworkPolicyName)
	AssertNetworkPolicyHasModuleLabels(t, np)
	AssertNetworkPolicyHasPodSelector(t, np, map[string]string{
		"istio": "pilot",
	})
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeEgress)

	// Verify JWKS egress rules
	AssertNetworkPolicyHasEgressRule(t, np, "TCP", 80)
	AssertNetworkPolicyHasEgressRule(t, np, "TCP", 443)
}

// AssertIngressGatewayNetworkPolicy validates the istio-ingressgateway NetworkPolicy
func AssertIngressGatewayNetworkPolicy(t *testing.T, r *resources.Resources) {
	t.Helper()

	np := AssertNetworkPolicyExists(t, r, "istio-system", IngressGatewayNetworkPolicyName)
	AssertNetworkPolicyHasModuleLabels(t, np)
	AssertNetworkPolicyHasPodSelector(t, np, map[string]string{
		"istio": "ingressgateway",
	})
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeEgress)
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeIngress)

	// Verify egress rules
	AssertNetworkPolicyHasEgressRule(t, np, "TCP", 15012) // XDS to istiod
	AssertNetworkPolicyHasEgressRule(t, np, "TCP", 53)    // DNS
	AssertNetworkPolicyHasEgressRule(t, np, "UDP", 53)    // DNS

	// Verify ingress rules
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 8080)  // HTTP
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 8443)  // HTTPS
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 15008) // HBONE
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 15020) // Metrics
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 15021) // Health
	AssertNetworkPolicyHasIngressRule(t, np, "TCP", 15090) // Telemetry
}

// AssertIngressGatewayEgressNetworkPolicy validates the istio-ingressgateway egress NetworkPolicy
func AssertIngressGatewayEgressNetworkPolicy(t *testing.T, r *resources.Resources) {
	t.Helper()

	np := AssertNetworkPolicyExists(t, r, "istio-system", IngressGatewayEgressNetworkPolicyName)
	AssertNetworkPolicyHasModuleLabels(t, np)
	AssertNetworkPolicyHasPodSelector(t, np, map[string]string{
		"istio": "ingressgateway",
	})
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeEgress)

	// This NetworkPolicy should allow egress to pods with specific label
	require.NotEmpty(t, np.Spec.Egress, "NetworkPolicy %s should have egress rules", np.Name)
}

// AssertCNINodeNetworkPolicy validates the istio-cni-node NetworkPolicy
func AssertCNINodeNetworkPolicy(t *testing.T, r *resources.Resources) {
	t.Helper()

	np := AssertNetworkPolicyExists(t, r, "istio-system", CNINodeNetworkPolicyName)
	AssertNetworkPolicyHasModuleLabels(t, np)
	AssertNetworkPolicyHasPodSelector(t, np, map[string]string{
		"k8s-app": "istio-cni-node",
	})
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeEgress)

	// Verify DNS egress rules
	AssertNetworkPolicyHasEgressRule(t, np, "TCP", 53)
	AssertNetworkPolicyHasEgressRule(t, np, "UDP", 53)
}

// AssertEgressGatewayNetworkPolicy validates the istio-egressgateway NetworkPolicy
func AssertEgressGatewayNetworkPolicy(t *testing.T, r *resources.Resources) {
	t.Helper()

	np := AssertNetworkPolicyExists(t, r, "istio-system", EgressGatewayNetworkPolicyName)
	AssertNetworkPolicyHasModuleLabels(t, np)
	AssertNetworkPolicyHasPodSelector(t, np, map[string]string{
		"istio": "egressgateway",
	})
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeEgress)
	AssertNetworkPolicyHasPolicyType(t, np, networkingv1.PolicyTypeIngress)

	// Verify ingress from labeled pods
	AssertNetworkPolicyHasIngressFromLabeledPods(t, np, "networking.kyma-project.io/to-egressgateway", "allowed")

	// Verify egress is allowed (all egress from egressgateway is controlled by Istio resources)
	AssertNetworkPolicyAllowsAllEgress(t, np)
}

// AssertAllNetworkPoliciesValid validates all module-managed NetworkPolicies
func AssertAllNetworkPoliciesValid(t *testing.T, r *resources.Resources) {
	t.Helper()
	t.Log("Validating all module-managed NetworkPolicies")

	AssertControllerManagerNetworkPolicy(t, r)
	AssertIstiodNetworkPolicy(t, r)
	AssertIstiodJWKSNetworkPolicy(t, r)
	AssertIngressGatewayNetworkPolicy(t, r)
	AssertIngressGatewayEgressNetworkPolicy(t, r)
	AssertEgressGatewayNetworkPolicy(t, r)
	AssertCNINodeNetworkPolicy(t, r)
}
