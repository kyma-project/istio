package prometheusmerge

import (
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	v1 "k8s.io/api/core/v1"
)

type ProxyRestartPredicate struct {
	oldPrometheusMerge bool
	newPrometheusMerge bool
}

func NewRestartPredicate(istioCR *v1alpha2.Istio) (*ProxyRestartPredicate, error) {
	lastAppliedConfig, err := istio.GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return nil, err
	}

	return &ProxyRestartPredicate{
		oldPrometheusMerge: lastAppliedConfig.IstioSpec.Config.Telemetry.Metrics.PrometheusMerge,
		newPrometheusMerge: istioCR.Spec.Config.Telemetry.Metrics.PrometheusMerge,
	}, nil
}

func (p ProxyRestartPredicate) RequiresProxyRestart(_ v1.Pod) bool {
	return p.oldPrometheusMerge != p.newPrometheusMerge
}
