package networkpolicy

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/yaml"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

// NetworkPolicyBuilder provides a fluent API for building and deploying NetworkPolicies
type NetworkPolicyBuilder struct {
	np *networkingv1.NetworkPolicy
}

// NewNetworkPolicyBuilder creates a new NetworkPolicy builder with default values
func NewNetworkPolicyBuilder(name, namespace string) *NetworkPolicyBuilder {
	return &NetworkPolicyBuilder{
		np: &networkingv1.NetworkPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: networkingv1.NetworkPolicySpec{
				PodSelector: metav1.LabelSelector{},
			},
		},
	}
}

// WithPodSelector sets the pod selector for the NetworkPolicy
func (b *NetworkPolicyBuilder) WithPodSelector(labels map[string]string) *NetworkPolicyBuilder {
	b.np.Spec.PodSelector = metav1.LabelSelector{
		MatchLabels: labels,
	}
	return b
}

// WithLabel adds a label to the NetworkPolicy
func (b *NetworkPolicyBuilder) WithLabel(key, value string) *NetworkPolicyBuilder {
	if b.np.ObjectMeta.Labels == nil {
		b.np.ObjectMeta.Labels = make(map[string]string)
	}
	b.np.ObjectMeta.Labels[key] = value
	return b
}

// WithPolicyTypes sets the policy types (Ingress, Egress)
func (b *NetworkPolicyBuilder) WithPolicyTypes(policyTypes ...networkingv1.PolicyType) *NetworkPolicyBuilder {
	b.np.Spec.PolicyTypes = policyTypes
	return b
}

// WithIngressRule adds an ingress rule to the NetworkPolicy
func (b *NetworkPolicyBuilder) WithIngressRule(rule networkingv1.NetworkPolicyIngressRule) *NetworkPolicyBuilder {
	b.np.Spec.Ingress = append(b.np.Spec.Ingress, rule)
	return b
}

// WithEgressRule adds an egress rule to the NetworkPolicy
func (b *NetworkPolicyBuilder) WithEgressRule(rule networkingv1.NetworkPolicyEgressRule) *NetworkPolicyBuilder {
	b.np.Spec.Egress = append(b.np.Spec.Egress, rule)
	return b
}

// Build returns the constructed NetworkPolicy
func (b *NetworkPolicyBuilder) Build() *networkingv1.NetworkPolicy {
	return b.np
}

// DeployWithCleanup deploys the NetworkPolicy and registers cleanup
func (b *NetworkPolicyBuilder) DeployWithCleanup(t *testing.T, r *resources.Resources) (*networkingv1.NetworkPolicy, error) {
	t.Helper()

	np := b.Build()

	ym, err := yaml.Marshal(np)
	if err != nil {
		t.Logf("Failed to marshal NetworkPolicy %s/%s to YAML: %v", np.Namespace, np.Name, err)
		return nil, err
	}
	t.Logf("Deploying NetworkPolicy %s/%s:\n%s", np.Namespace, np.Name, string(ym))

	err = r.Create(context.Background(), np)
	if err != nil {
		t.Logf("Failed to create NetworkPolicy %s/%s: %v", np.Namespace, np.Name, err)
		return nil, err
	}

	t.Logf("Created NetworkPolicy %s/%s", np.Namespace, np.Name)

	setup.DeclareCleanup(t, func() {
		t.Logf("Cleaning up NetworkPolicy %s/%s", np.Namespace, np.Name)
		err := r.Delete(setup.GetCleanupContext(), np)
		if err != nil {
			t.Logf("Failed to delete NetworkPolicy %s/%s: %v", np.Namespace, np.Name, err)
		}
	})

	return np, nil
}

// Helper functions for creating common NetworkPolicy rules

// NewIngressRuleFromNamespace creates an ingress rule allowing traffic from a specific namespace
func NewIngressRuleFromNamespace(namespace string, ports ...int32) networkingv1.NetworkPolicyIngressRule {
	rule := networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubernetes.io/metadata.name": namespace,
					},
				},
			},
		},
	}

	for _, port := range ports {
		portValue := intstr.FromInt32(port)
		protocol := corev1.ProtocolTCP
		rule.Ports = append(rule.Ports, networkingv1.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &portValue,
		})
	}

	return rule
}

// NewIngressRuleFromPodSelector creates an ingress rule allowing traffic from pods matching labels in the same namespace
func NewIngressRuleFromPodSelector(labels map[string]string, ports ...int32) networkingv1.NetworkPolicyIngressRule {
	rule := networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{
			{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
			},
		},
	}

	for _, port := range ports {
		portValue := intstr.FromInt32(port)
		protocol := corev1.ProtocolTCP
		rule.Ports = append(rule.Ports, networkingv1.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &portValue,
		})
	}

	return rule
}

// NewIngressRuleFromPodSelectorAcrossNamespaces creates an ingress rule allowing traffic from pods matching labels in a specific namespace
func NewIngressRuleFromPodSelectorAcrossNamespaces(namespace string, labels map[string]string, ports ...int32) networkingv1.NetworkPolicyIngressRule {
	rule := networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubernetes.io/metadata.name": namespace,
					},
				},
				PodSelector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
			},
		},
	}

	for _, port := range ports {
		portValue := intstr.FromInt32(port)
		protocol := corev1.ProtocolTCP
		rule.Ports = append(rule.Ports, networkingv1.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &portValue,
		})
	}

	return rule
}

// NewIngressRuleAllowAll creates an ingress rule allowing all traffic on specified ports
func NewIngressRuleAllowAll(ports ...int32) networkingv1.NetworkPolicyIngressRule {
	rule := networkingv1.NetworkPolicyIngressRule{
		From: []networkingv1.NetworkPolicyPeer{},
	}

	for _, port := range ports {
		portValue := intstr.FromInt32(port)
		protocol := corev1.ProtocolTCP
		rule.Ports = append(rule.Ports, networkingv1.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &portValue,
		})
	}

	return rule
}

// NewEgressRuleToNamespace creates an egress rule allowing traffic to a specific namespace
func NewEgressRuleToNamespace(namespace string, ports ...int32) networkingv1.NetworkPolicyEgressRule {
	rule := networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{
			{
				NamespaceSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"kubernetes.io/metadata.name": namespace,
					},
				},
			},
		},
	}

	for _, port := range ports {
		portValue := intstr.FromInt32(port)
		protocol := corev1.ProtocolTCP
		rule.Ports = append(rule.Ports, networkingv1.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &portValue,
		})
	}

	return rule
}

// NewEgressRuleToPodSelector creates an egress rule allowing traffic to pods matching labels
func NewEgressRuleToPodSelector(labels map[string]string, ports ...int32) networkingv1.NetworkPolicyEgressRule {
	rule := networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{
			{
				PodSelector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
			},
		},
	}

	for _, port := range ports {
		portValue := intstr.FromInt32(port)
		protocol := corev1.ProtocolTCP
		rule.Ports = append(rule.Ports, networkingv1.NetworkPolicyPort{
			Protocol: &protocol,
			Port:     &portValue,
		})
	}

	return rule
}

// NewEgressRuleDNS creates an egress rule for DNS (TCP and UDP port 53) to kube-system
func NewEgressRuleDNS() networkingv1.NetworkPolicyEgressRule {
	tcpProtocol := corev1.ProtocolTCP
	udpProtocol := corev1.ProtocolUDP
	port := intstr.FromInt32(53)

	return networkingv1.NetworkPolicyEgressRule{
		Ports: []networkingv1.NetworkPolicyPort{
			{
				Protocol: &tcpProtocol,
				Port:     &port,
			},
			{
				Protocol: &udpProtocol,
				Port:     &port,
			},
		},
	}
}

// NewEgressRuleAllowAll creates an egress rule allowing all traffic
func NewEgressRuleAllowAll() networkingv1.NetworkPolicyEgressRule {
	return networkingv1.NetworkPolicyEgressRule{
		To: []networkingv1.NetworkPolicyPeer{},
	}
}

// Preset NetworkPolicy builders for common scenarios

// NewDefaultDenyNetworkPolicy creates a default-deny NetworkPolicy
func NewDefaultDenyNetworkPolicy(namespace string) *NetworkPolicyBuilder {
	return NewNetworkPolicyBuilder("default-deny-all", namespace).
		WithPodSelector(map[string]string{}). // Empty selector = all pods
		WithPolicyTypes(networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress)
}

// NewAllowFromIngressGatewayNetworkPolicy creates a NetworkPolicy allowing traffic from istio-ingressgateway
func NewAllowFromIngressGatewayNetworkPolicy(namespace string, targetPodLabels map[string]string) *NetworkPolicyBuilder {
	ingressRule := NewIngressRuleFromPodSelectorAcrossNamespaces(
		"istio-system",
		map[string]string{"istio": "ingressgateway"},
		8080, 8443, 15021,
	)

	return NewNetworkPolicyBuilder("allow-from-ingressgateway", namespace).
		WithPodSelector(targetPodLabels).
		WithPolicyTypes(networkingv1.PolicyTypeIngress).
		WithIngressRule(ingressRule)
}

// NewAllowToIstioSystemNetworkPolicy creates a NetworkPolicy allowing egress to istio-system
func NewAllowToIstioSystemNetworkPolicy(namespace string, sourcePodLabels map[string]string) *NetworkPolicyBuilder {
	egressRule := NewEgressRuleToNamespace("istio-system", 15012, 15014, 15017)

	return NewNetworkPolicyBuilder("allow-to-istio-system", namespace).
		WithPodSelector(sourcePodLabels).
		WithPolicyTypes(networkingv1.PolicyTypeEgress).
		WithEgressRule(egressRule).
		WithEgressRule(NewEgressRuleDNS())
}

// NewAllowFromIstiodNetworkPolicy creates a NetworkPolicy allowing ingress traffic from istiod (istio-pilot) in istio-system namespace
// This is required when using module NetworkPolicies and an in-cluster JWKS endpoint, as istiod needs to fetch JWKS from the mock server
func NewAllowFromIstiodNetworkPolicy(namespace string, targetPodLabels map[string]string, ports ...int32) *NetworkPolicyBuilder {
	ingressRule := NewIngressRuleFromPodSelectorAcrossNamespaces(
		"istio-system",
		map[string]string{"istio": "pilot"},
		ports...,
	)

	return NewNetworkPolicyBuilder("allow-from-istiod", namespace).
		WithPodSelector(targetPodLabels).
		WithPolicyTypes(networkingv1.PolicyTypeIngress).
		WithIngressRule(ingressRule)
}
