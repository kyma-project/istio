package v2

import (
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

func TestDNSProxyingChanged_Evaluate(t *testing.T) {
	tc := []struct {
		name           string
		old            *bool
		current        *bool
		expectDecision Decision
	}{
		{name: "both nil, continue", old: nil, current: nil, expectDecision: Continue},
		{name: "same value, continue", old: new(true), current: new(true), expectDecision: Continue},
		{name: "value changed, restart", old: new(false), current: new(true), expectDecision: Restart},
		{name: "nil to set, restart", old: nil, current: new(true), expectDecision: Restart},
		{name: "set to nil, restart", old: new(true), current: nil, expectDecision: Restart},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cr := operatorv1alpha2.Istio{}
			cr.Spec.Config.EnableDNSProxying = tt.current
			lastApplied := configuration.AppliedConfig{}
			lastApplied.Config.EnableDNSProxying = tt.old

			eval := NewDNSProxyingChanged(cr, lastApplied)
			d := eval.Evaluate(nil)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}
