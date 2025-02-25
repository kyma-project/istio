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

func (p PrometheusMergeRestartPredicate) Matches(pod v1.Pod) bool {
	// No change in configuration, no restart needed
	if p.oldPrometheusMerge == p.newPrometheusMerge {
		return false
	}

	annotations := pod.GetAnnotations()
	const (
		prometheusMergePath = "/stats/prometheus"
		prometheusMergePort = "15020"
	)

	hasPrometheusMergePath := annotations["prometheus.io/path"] == prometheusMergePath
	hasPrometheusMergePort := annotations["prometheus.io/port"] == prometheusMergePort

	// When enabling PrometheusMerge, restart if prometheusMerge annotations are missing or incorrect
	if p.newPrometheusMerge {
		return !hasPrometheusMergePath || !hasPrometheusMergePort
	}

	// When disabling PrometheusMerge, restart if prometheusMerge annotations are present and correct
	return hasPrometheusMergePath || hasPrometheusMergePort
}

func (p PrometheusMergeRestartPredicate) MustMatch() bool {
	return false
}
