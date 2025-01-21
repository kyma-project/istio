package sidecars

import (
	"context"

	"github.com/kyma-project/istio/operator/internal/filter"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	podsToRestartLimit = 30
	podsToListLimit    = 100
)

type ProxyResetter interface {
	ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, logger *logr.Logger) ([]restart.RestartWarning, bool)
}

type ProxyReset struct {
}

func NewProxyResetter() *ProxyReset {
	return &ProxyReset{}
}

func (p *ProxyReset) ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, logger *logr.Logger) ([]restart.RestartWarning, bool) {
	limits := pods.NewPodsRestartLimits(podsToRestartLimit, podsToListLimit)
	podsToRestart, err := pods.GetPodsToRestart(ctx, c, expectedImage, expectedResources, predicates, limits, logger)
	if err != nil {
		logger.Error(err, "Getting pods to restart failed")
	}

	// if there are more pods to restart there should be a continue token in the pod list
	hasMorePodsToRestart := podsToRestart.Continue != ""
	warnings := restart.Restart(ctx, c, podsToRestart, logger)

	if !hasMorePodsToRestart {
		logger.Info("Proxy reset completed")
	} else {
		logger.Info("Proxy reset only partially completed")
	}

	return warnings, hasMorePodsToRestart
}
