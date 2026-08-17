package predicates

import (
	"slices"

	v1 "k8s.io/api/core/v1"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

type ProxyStatsMatcherRestartPredicate struct {
	oldInclusionRegexps []string
	newInclusionRegexps []string
}

func NewProxyStatsMatcherRestartPredicate(istioCR *v1alpha2.Istio) (*ProxyStatsMatcherRestartPredicate, error) {
	lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return nil, err
	}

	var oldRegexps []string
	if lastAppliedConfig.Config.ProxyStatsMatcher != nil {
		oldRegexps = lastAppliedConfig.Config.ProxyStatsMatcher.InclusionRegexps
	}

	var newRegexps []string
	if istioCR.Spec.Config.ProxyStatsMatcher != nil {
		newRegexps = istioCR.Spec.Config.ProxyStatsMatcher.InclusionRegexps
	}

	return &ProxyStatsMatcherRestartPredicate{
		oldInclusionRegexps: oldRegexps,
		newInclusionRegexps: newRegexps,
	}, nil
}

func (p ProxyStatsMatcherRestartPredicate) Matches(_ v1.Pod) bool {
	return !slices.Equal(p.oldInclusionRegexps, p.newInclusionRegexps)
}

func (p ProxyStatsMatcherRestartPredicate) MustMatch() bool {
	return false
}

func (p ProxyStatsMatcherRestartPredicate) Name() string {
	return "ProxyStatsMatcherRestartPredicate"
}
