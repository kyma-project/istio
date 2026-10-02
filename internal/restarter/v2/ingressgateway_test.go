package v2

import (
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ingressGatewayDeployment() client.Object {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "istio-ingressgateway", Namespace: "istio-system"},
	}
}

func TestNumTrustedProxiesConfigChanged_Evaluate(t *testing.T) {
	tc := []struct {
		name           string
		old            *int
		current        *int
		object         client.Object
		expectDecision Decision
	}{
		{name: "both nil, continue", old: nil, current: nil, object: ingressGatewayDeployment(), expectDecision: Continue},
		{name: "same value, continue", old: new(1), current: new(1), object: ingressGatewayDeployment(), expectDecision: Continue},
		{name: "value changed, restart", old: new(1), current: new(2), object: ingressGatewayDeployment(), expectDecision: Restart},
		{name: "nil to set, restart", old: nil, current: new(1), object: ingressGatewayDeployment(), expectDecision: Restart},
		{name: "set to nil, restart", old: new(1), current: nil, object: ingressGatewayDeployment(), expectDecision: Restart},
		{
			name: "value changed but not ingressgateway deployment, continue",
			old:  new(1), current: new(2),
			object:         &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "istio-system"}},
			expectDecision: Continue,
		},
		{
			name: "value changed but not a deployment, continue",
			old:  new(1), current: new(2),
			object:         &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "istio-ingressgateway", Namespace: "istio-system"}},
			expectDecision: Continue,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cr := operatorv1alpha2.Istio{}
			cr.Spec.Config.NumTrustedProxies = tt.current
			lastApplied := configuration.AppliedConfig{}
			lastApplied.Config.NumTrustedProxies = tt.old

			eval := NewNumTrustedProxiesConfigChanged(cr, lastApplied)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}

func TestForwardClientCertDetailsConfigChanged_Evaluate(t *testing.T) {
	tc := []struct {
		name           string
		old            *operatorv1alpha2.XFCCStrategy
		current        *operatorv1alpha2.XFCCStrategy
		object         client.Object
		expectDecision Decision
	}{
		{name: "both nil, continue", old: nil, current: nil, object: ingressGatewayDeployment(), expectDecision: Continue},
		{name: "same value, continue", old: new(operatorv1alpha2.SanitizeSet), current: new(operatorv1alpha2.SanitizeSet), object: ingressGatewayDeployment(), expectDecision: Continue},
		{name: "value changed, restart", old: new(operatorv1alpha2.SanitizeSet), current: new(operatorv1alpha2.ForwardOnly), object: ingressGatewayDeployment(), expectDecision: Restart},
		{name: "nil to set, restart", old: nil, current: new(operatorv1alpha2.SanitizeSet), object: ingressGatewayDeployment(), expectDecision: Restart},
		{name: "set to nil, restart", old: new(operatorv1alpha2.SanitizeSet), current: nil, object: ingressGatewayDeployment(), expectDecision: Restart},
		{
			name: "value changed but not ingressgateway deployment, continue",
			old:  new(operatorv1alpha2.SanitizeSet), current: new(operatorv1alpha2.ForwardOnly),
			object:         &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "istio-system"}},
			expectDecision: Continue,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cr := operatorv1alpha2.Istio{}
			cr.Spec.Config.ForwardClientCertDetails = tt.current
			lastApplied := configuration.AppliedConfig{}
			lastApplied.Config.ForwardClientCertDetails = tt.old

			eval := NewForwardClientCertDetailsConfigChanged(cr, lastApplied)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}

func TestTrustDomainConfigChanged_Evaluate(t *testing.T) {
	tc := []struct {
		name           string
		old            *string
		current        *string
		object         client.Object
		expectDecision Decision
	}{
		{name: "both nil, continue", old: nil, current: nil, object: ingressGatewayDeployment(), expectDecision: Continue},
		{name: "same value, continue", old: new("cluster.local"), current: new("cluster.local"), object: ingressGatewayDeployment(), expectDecision: Continue},
		{name: "value changed, restart", old: new("cluster.local"), current: new("custom.domain"), object: ingressGatewayDeployment(), expectDecision: Restart},
		{name: "nil to set, restart", old: nil, current: new("cluster.local"), object: ingressGatewayDeployment(), expectDecision: Restart},
		{name: "set to nil, restart", old: new("cluster.local"), current: nil, object: ingressGatewayDeployment(), expectDecision: Restart},
		{
			name: "value changed but not ingressgateway deployment, continue",
			old:  new("cluster.local"), current: new("custom.domain"),
			object:         &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "istio-system"}},
			expectDecision: Continue,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cr := operatorv1alpha2.Istio{}
			cr.Spec.Config.TrustDomain = tt.current
			lastApplied := configuration.AppliedConfig{}
			lastApplied.Config.TrustDomain = tt.old

			eval := NewTrustDomainConfigChanged(cr, lastApplied)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}
