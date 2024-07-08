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

type ProxyResetter interface {
	ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, logger *logr.Logger) ([]restart.RestartWarning, bool, error)
}

type ProxyReset struct {
}

func NewProxyResetter() *ProxyReset {
	return &ProxyReset{}
}

func (p *ProxyReset) ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, logger *logr.Logger) ([]restart.RestartWarning, bool, error) {
	podListToRestart, err := pods.GetPodsToRestart(ctx, c, expectedImage, expectedResources, predicates, logger)
	if err != nil {
		return nil, false, err
	}

	// if there are more pods to restart there should be a continue token in the pod list
	hasMorePodsToRestart := podListToRestart.Continue != ""
	warnings, err := restart.Restart(ctx, c, podListToRestart, logger)
	if err != nil {
		return nil, hasMorePodsToRestart, err
	}

	if !hasMorePodsToRestart {
		logger.Info("Proxy reset completed")
	} else {
		leftoverPodsToRestart := int64(0)
		if podListToRestart.RemainingItemCount != nil {
			leftoverPodsToRestart = *podListToRestart.RemainingItemCount
		}
		logger.Info("Proxy reset partially completed", "count of leftover pods", leftoverPodsToRestart)
	}

	return warnings, hasMorePodsToRestart, nil
}
