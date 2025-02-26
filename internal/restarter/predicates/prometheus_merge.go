package predicates

import (
	"context"
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	"istio.io/istio/pkg/config/mesh"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

const defaultStatusPort int32 = 15020

type PrometheusMergeRestartPredicate struct {
	oldPrometheusMerge bool
	newPrometheusMerge bool
	statusPort         string
}

func NewPrometheusMergeRestartPredicate(ctx context.Context, client client.Client, istioCR *v1alpha2.Istio) (*PrometheusMergeRestartPredicate, error) {
	lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return nil, err
	}

	statusPort := getStatusPort(ctx, client)

	return &PrometheusMergeRestartPredicate{
		oldPrometheusMerge: lastAppliedConfig.IstioSpec.Config.Telemetry.Metrics.PrometheusMerge,
		newPrometheusMerge: istioCR.Spec.Config.Telemetry.Metrics.PrometheusMerge,
		statusPort:         strconv.FormatInt(int64(statusPort), 10),
	}, nil
}

func (p PrometheusMergeRestartPredicate) Matches(pod v1.Pod) bool {
	// No change in configuration, no restart needed
	if p.oldPrometheusMerge == p.newPrometheusMerge {
		return false
	}

	annotations := pod.GetAnnotations()
	var (
		prometheusMergePath = "/stats/prometheus"
		prometheusMergePort = p.statusPort
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

// Gets statusPort directly from already merged IstioOperator CR, for now it is 15020 by default and not configurable,
// but once it is configurable, it will fetch the configured statusPort from the CR directly
func getStatusPort(ctx context.Context, client client.Client) int32 {
	istioConfigMap := &v1.ConfigMap{}

	var err = client.Get(ctx, types.NamespacedName{Namespace: "istio-system", Name: "istio"}, istioConfigMap)
	if err != nil {
		return defaultStatusPort
	}

	meshConfigYAML, hasMesh := istioConfigMap.Data["mesh"]
	if !hasMesh {
		return defaultStatusPort
	}

	// Clean up the YAML string - remove any leading indicators like "|-"
	meshConfigYAML = strings.TrimPrefix(meshConfigYAML, "|-")
	meshConfigYAML = strings.TrimSpace(meshConfigYAML)

	meshConfig, err := mesh.ApplyMeshConfigDefaults(meshConfigYAML)

	if err != nil {
		return defaultStatusPort
	}

	if meshConfig.DefaultConfig.StatusPort == 0 {
		return defaultStatusPort

	}

	return meshConfig.DefaultConfig.StatusPort
}
