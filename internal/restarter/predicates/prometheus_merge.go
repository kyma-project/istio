package predicates

import (
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	v1 "k8s.io/api/core/v1"
)

type PrometheusMergeRestartPredicate struct {
	oldPrometheusMerge bool
	newPrometheusMerge bool
}

func NewPrometheusMergeRestartPredicate(istioCR *v1alpha2.Istio) (*PrometheusMergeRestartPredicate, error) {
	lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return nil, err
	}

	return &PrometheusMergeRestartPredicate{
		oldPrometheusMerge: lastAppliedConfig.IstioSpec.Config.Telemetry.Metrics.PrometheusMerge,
		newPrometheusMerge: istioCR.Spec.Config.Telemetry.Metrics.PrometheusMerge,
	}, nil
}

func (p PrometheusMergeRestartPredicate) Matches(_ v1.Pod) bool {
	return p.oldPrometheusMerge != p.newPrometheusMerge
}

func (p PrometheusMergeRestartPredicate) MustMatch() bool {
	return false
}
