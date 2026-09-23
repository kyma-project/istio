package v2

import (
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

func TestCompatibilityModeChanged_Evaluate(t *testing.T) {
	// ProxyMetaDataCompatibility is currently empty, so the constructor always sets
	// hasProxyMetadata to false; the hasProxyMetadata case is built directly until
	// the real hashed-state logic lands.
	tc := []struct {
		name             string
		old              bool
		current          bool
		hasProxyMetadata bool
		expectDecision   Decision
	}{
		{name: "mode unchanged, continue", old: false, current: false, expectDecision: Continue},
		{name: "mode changed but no proxy metadata, continue", old: false, current: true, expectDecision: Continue},
		{name: "mode changed with proxy metadata, restart", old: false, current: true, hasProxyMetadata: true, expectDecision: Restart},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cr := operatorv1alpha2.Istio{}
			cr.Spec.CompatibilityMode = tt.current
			lastApplied := configuration.AppliedConfig{}
			lastApplied.CompatibilityMode = tt.old
			testMeta := map[string]string{}
			if tt.hasProxyMetadata {
				testMeta = map[string]string{"FOO": "true"}
			}
			eval := NewCompatibilityModeChanged(cr, lastApplied, testMeta)
			d := eval.Evaluate(nil)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}
