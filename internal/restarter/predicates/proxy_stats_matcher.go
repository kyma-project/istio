package predicates

import (
	"context"
	"slices"

	v1 "k8s.io/api/core/v1"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

type ProxyStatsMatcherRestartPredicate struct {
	oldInclusionRegexps []string
	newInclusionRegexps []string
}

func NewProxyStatsMatcherRestartPredicate(istioCR *v1alpha2.Istio, lastAppliedConfig configuration.AppliedConfig) *ProxyStatsMatcherRestartPredicate {
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

	return &ProxyStatsMatcherRestartPredicate{
		oldInclusionRegexps: oldRegexps,
		newInclusionRegexps: newRegexps,
	}
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

func (p ProxyStatsMatcherRestartPredicate) NewIngressGatewayEvaluator(_ context.Context) (IngressGatewayRestartEvaluator, error) {
	return p, nil
}

func (p ProxyStatsMatcherRestartPredicate) RequiresIngressGatewayRestart() bool {
	return !slices.Equal(p.oldInclusionRegexps, p.newInclusionRegexps)
}
