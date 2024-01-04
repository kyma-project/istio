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
	ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, logger *logr.Logger) ([]restart.RestartWarning, error)
}

type ProxyReset struct {
}

func NewProxyResetter() *ProxyReset {
	return &ProxyReset{}
}

func (p *ProxyReset) ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, logger *logr.Logger) ([]restart.RestartWarning, error) {
	podListToRestart, err := pods.GetPodsToRestart(ctx, c, expectedImage, expectedResources, predicates, logger)
	if err != nil {
		return nil, err
	}

	warnings, err := restart.Restart(ctx, c, podListToRestart, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("Proxy reset done")

	return warnings, nil
}
