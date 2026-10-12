package v2

import (
	"slices"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

// ProxyStatsMatcherConfigChanged is an Evaluator that detects a change in the
// Istio proxy stats matcher inclusion regexps between the last applied config and
// the current CR.
type ProxyStatsMatcherConfigChanged struct {
	oldInclusionRegexps []string
	newInclusionRegexps []string
}

// Evaluate returns Restart when the inclusion regexps changed, and Continue
// otherwise.
func (p *ProxyStatsMatcherConfigChanged) Evaluate(_ Object) Decision {
	if !slices.Equal(p.oldInclusionRegexps, p.newInclusionRegexps) {
		return Restart
	}
	return Continue
}

// NewProxyStatsMatcherConfigChanged returns an Evaluator that restarts workloads
// when the proxy stats matcher inclusion regexps changed. The regexps are sorted
// so that ordering differences alone do not trigger a restart.
func NewProxyStatsMatcherConfigChanged(istioCR *v1alpha2.Istio, lastAppliedConfig configuration.AppliedConfig) Evaluator {
	var oldRegexps []string
	if lastAppliedConfig.Config.ProxyStatsMatcher != nil {
		oldRegexps = lastAppliedConfig.Config.ProxyStatsMatcher.InclusionRegexps
	}
	slices.Sort(oldRegexps)

	var newRegexps []string
	if istioCR.Spec.Config.ProxyStatsMatcher != nil {
		newRegexps = istioCR.Spec.Config.ProxyStatsMatcher.InclusionRegexps
	}
	slices.Sort(newRegexps)

	return &ProxyStatsMatcherConfigChanged{
		oldInclusionRegexps: oldRegexps,
		newInclusionRegexps: newRegexps,
	}
}
