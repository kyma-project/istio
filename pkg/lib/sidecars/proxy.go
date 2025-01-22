package sidecars

import (
	"context"
	"math"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"

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

type ProxyRestarter interface {
	RestartProxies(ctx context.Context, c client.Client, expectedImage predicates.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio, logger *logr.Logger) ([]restart.RestartWarning, bool, error)
}

type ProxyRestart struct {
}

func NewProxyRestarter() *ProxyRestart {
	return &ProxyRestart{}
}

func (p *ProxyRestart) RestartProxies(ctx context.Context, c client.Client, expectedImage predicates.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio, logger *logr.Logger) ([]restart.RestartWarning, bool, error) {
	compatibiltyPredicate, err := predicates.NewCompatibilityRestartPredicate(istioCR)
	if err != nil {
		logger.Error(err, "Failed to create restart compatibility predicate")
		return nil, false, err
	}

	predicates := []predicates.SidecarProxyPredicate{compatibiltyPredicate,
		predicates.NewImageResourcesPredicate(expectedImage, expectedResources),
	}

	err = p.resetKymaProxies(ctx, c, predicates, logger)
	if err != nil {
		logger.Error(err, "Failed to restart Kyma proxies")
		return nil, false, err
	}

	warnings, hasMorePodsToRestart, err := p.resetCustomerProxies(ctx, c, predicates, logger)
	if err != nil {
		logger.Error(err, "Failed to restart Customer proxies")
	}

	return warnings, hasMorePodsToRestart, nil
}

func (p *ProxyRestart) resetKymaProxies(ctx context.Context, c client.Client, preds []predicates.SidecarProxyPredicate, logger *logr.Logger) error {
	preds = append(preds, predicates.KymaWorkloadRestartPredicate{})
	limits := pods.NewPodsRestartLimits(math.MaxInt, math.MaxInt)

	_, _, err := p.restart(ctx, c, preds, limits, logger)
	if err != nil {
		logger.Error(err, "Failed to restart Kyma proxies")
		return err
	}

	logger.Info("Kyma proxy reset completed")
	return nil
}

func (p *ProxyRestart) resetCustomerProxies(ctx context.Context, c client.Client, preds []predicates.SidecarProxyPredicate, logger *logr.Logger) ([]restart.RestartWarning, bool, error) {
	preds = append(preds, predicates.CustomerWorkloadRestartPredicate{})
	limits := pods.NewPodsRestartLimits(podsToRestartLimit, podsToListLimit)

	warnings, hasMorePodsToRestart, err := p.restart(ctx, c, preds, limits, logger)
	if err != nil {
		logger.Error(err, "Failed to restart Customer proxies")
		return nil, false, err
	}

	return warnings, hasMorePodsToRestart, nil
}

func (p *ProxyRestart) restart(ctx context.Context, c client.Client, preds []predicates.SidecarProxyPredicate, limits *pods.PodsRestartLimits, logger *logr.Logger) ([]restart.RestartWarning, bool, error) {
	podsToRestart, err := pods.GetPodsToRestart(ctx, c, preds, limits, logger)
	if err != nil {
		logger.Error(err, "Getting Kyma pods to restart failed")
		return nil, false, err
	}

	warnings, err := restart.Restart(ctx, c, podsToRestart, logger, true)
	if err != nil {
		logger.Error(err, "Restarting Kyma pods failed")
		return nil, false, err
	}

	// if there are more pods to restart there should be a continue token in the pod list
	hasMorePodsToRestart := podsToRestart.Continue != ""

	if !hasMorePodsToRestart {
		logger.Info("Proxy reset completed")
	} else {
		logger.Info("Proxy reset only partially completed")
	}

	return warnings, hasMorePodsToRestart, nil
}
