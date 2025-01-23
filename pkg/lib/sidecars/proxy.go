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
	RestartProxies(ctx context.Context, expectedImage predicates.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio) ([]restart.RestartWarning, bool, error)
	RestartWithPredicates(ctx context.Context, preds []predicates.SidecarProxyPredicate, limits *pods.PodsRestartLimits, failOnError bool) ([]restart.RestartWarning, bool, error)
}

type ProxyRestart struct {
	k8sClient  client.Client
	podsLister pods.PodsGetter
	logger     *logr.Logger
}

func NewProxyRestarter(c client.Client, podsLister pods.PodsGetter, logger *logr.Logger) *ProxyRestart {
	return &ProxyRestart{
		k8sClient:  c,
		podsLister: podsLister,
		logger:     logger,
	}
}

func (p *ProxyRestart) RestartProxies(ctx context.Context, expectedImage predicates.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio) ([]restart.RestartWarning, bool, error) {
	compatibiltyPredicate, err := predicates.NewCompatibilityRestartPredicate(istioCR)
	if err != nil {
		p.logger.Error(err, "Failed to create restart compatibility predicate")
		return nil, false, err
	}

	predicates := []predicates.SidecarProxyPredicate{compatibiltyPredicate,
		predicates.NewImageResourcesPredicate(expectedImage, expectedResources),
	}

	err = p.restartKymaProxies(ctx, predicates)
	if err != nil {
		p.logger.Error(err, "Failed to restart Kyma proxies")
		return nil, false, err
	}

	warnings, hasMorePodsToRestart, err := p.restartCustomerProxies(ctx, predicates)
	if err != nil {
		p.logger.Error(err, "Failed to restart Customer proxies")
	}

	return warnings, hasMorePodsToRestart, nil
}

func (p *ProxyRestart) RestartWithPredicates(ctx context.Context, preds []predicates.SidecarProxyPredicate, limits *pods.PodsRestartLimits, failOnError bool) ([]restart.RestartWarning, bool, error) {
	podsToRestart, err := p.podsLister.GetPodsToRestart(ctx, preds, limits)
	if err != nil {
		p.logger.Error(err, "Getting pods to restart failed")
		return nil, false, err
	}

	warnings, err := restart.Restart(ctx, p.k8sClient, podsToRestart, p.logger, failOnError)
	if err != nil {
		p.logger.Error(err, "Restarting pods failed")
		return nil, false, err
	}

	// if there are more pods to restart there should be a continue token in the pod list
	return warnings, podsToRestart.Continue != "", nil
}

func (p *ProxyRestart) restartKymaProxies(ctx context.Context, preds []predicates.SidecarProxyPredicate) error {
	preds = append(preds, predicates.NewKymaWorkloadRestartPredicate())
	limits := pods.NewPodsRestartLimits(math.MaxInt, math.MaxInt)

	_, _, err := p.RestartWithPredicates(ctx, preds, limits, true)
	if err != nil {
		p.logger.Error(err, "Failed to restart Kyma proxies")
		return err
	}

	p.logger.Info("Kyma proxy restart completed")
	return nil
}

func (p *ProxyRestart) restartCustomerProxies(ctx context.Context, preds []predicates.SidecarProxyPredicate) ([]restart.RestartWarning, bool, error) {
	preds = append(preds, predicates.NewCustomerWorkloadRestartPredicate())
	limits := pods.NewPodsRestartLimits(podsToRestartLimit, podsToListLimit)

	warnings, hasMorePodsToRestart, err := p.RestartWithPredicates(ctx, preds, limits, false)
	if err != nil {
		p.logger.Error(err, "Failed to restart Customer proxies")
		return nil, false, err
	}

	if !hasMorePodsToRestart {
		p.logger.Info("Customer proxy restart completed")
	} else {
		p.logger.Info("Customer proxy restart only partially completed")
	}

	return warnings, hasMorePodsToRestart, nil
}
