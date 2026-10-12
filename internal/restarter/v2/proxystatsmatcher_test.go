package v2

import (
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

func TestProxyStatsMatcherConfigChanged_Evaluate(t *testing.T) {
	tc := []struct {
		name           string
		old            []string
		current        []string
		expectDecision Decision
	}{
		{name: "both unset, continue", old: nil, current: nil, expectDecision: Continue},
		{name: "same regexps, continue", old: []string{"a", "b"}, current: []string{"a", "b"}, expectDecision: Continue},
		{name: "same regexps different order, continue", old: []string{"a", "b"}, current: []string{"b", "a"}, expectDecision: Continue},
		{name: "different regexps, restart", old: []string{"a"}, current: []string{"b"}, expectDecision: Restart},
		{name: "regexps added, restart", old: nil, current: []string{"a"}, expectDecision: Restart},
		{name: "regexps removed, restart", old: []string{"a"}, current: nil, expectDecision: Restart},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			cr := &operatorv1alpha2.Istio{}
			if tt.current != nil {
				cr.Spec.Config.ProxyStatsMatcher = &operatorv1alpha2.ProxyStatsMatcher{InclusionRegexps: tt.current}
			}
			lastApplied := configuration.AppliedConfig{}
			if tt.old != nil {
				lastApplied.Config.ProxyStatsMatcher = &operatorv1alpha2.ProxyStatsMatcher{InclusionRegexps: tt.old}
			}

			eval := NewProxyStatsMatcherConfigChanged(cr, lastApplied)
			d := eval.Evaluate(nil)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}
