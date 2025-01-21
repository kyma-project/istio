package sidecars

import (
	"context"
	"math"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/filter"
	predicates "github.com/kyma-project/istio/operator/internal/restarter/predicates"

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
	ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio, logger *logr.Logger) ([]restart.RestartWarning, bool, error)
}

type ProxyReset struct {
}

func NewProxyResetter() *ProxyReset {
	return &ProxyReset{}
}

func (p *ProxyReset) ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio, logger *logr.Logger) ([]restart.RestartWarning, bool, error) {
	// _, _, err := p.resetKymaProxies(ctx, c, expectedImage, expectedResources, istioCR, logger)
	// if err != nil {
	// 	logger.Error(err, "Failed to restart Kyma proxies")
	// 	return nil, false, err
	// }

	warnings, hasMorePodsToRestart, err := p.resetCustomerProxies(ctx, c, expectedImage, expectedResources, istioCR, logger)
	if err != nil {
		logger.Error(err, "Failed to restart Customer proxies")
		return nil, false, err
	}

	return warnings, hasMorePodsToRestart, err
}

func (p *ProxyReset) resetKymaProxies(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio, logger *logr.Logger) ([]restart.RestartWarning, bool, error) {
	compatibiltyPredicate, err := predicates.NewCompatibilityRestartPredicate(istioCR)
	if err != nil {
		logger.Error(err, "Failed to create restart compatibility predicate")
		return nil, false, err
	}

	predicates := []filter.SidecarProxyPredicate{compatibiltyPredicate, predicates.KymaWorkloadRestartPredicate{}}
	limits := pods.NewPodsRestartLimits(math.MaxInt, math.MaxInt)

	podsToRestart, err := pods.GetPodsToRestart(ctx, c, expectedImage, expectedResources, predicates, limits, logger)
	if err != nil {
		logger.Error(err, "Getting Kyma pods to restart failed")
		return nil, false, err
	}

	warnings, err := restart.Restart(ctx, c, podsToRestart, logger, true)
	if err != nil {
		logger.Error(err, "Restarting Kyma pods failed")
		return nil, false, err
	}

	logger.Info("Kyma proxy reset completed")
	return warnings, false, nil
}

func (p *ProxyReset) resetCustomerProxies(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio, logger *logr.Logger) ([]restart.RestartWarning, bool, error) {
	compatibiltyPredicate, err := predicates.NewCompatibilityRestartPredicate(istioCR)
	if err != nil {
		logger.Error(err, "Failed to create restart compatibility predicate")
		return nil, false, err
	}

	predicates := []filter.SidecarProxyPredicate{compatibiltyPredicate /*, predicates.CustomerWorkloadRestartPredicate{}*/}
	limits := pods.NewPodsRestartLimits(podsToRestartLimit, podsToListLimit)

	podsToRestart, err := pods.GetPodsToRestart(ctx, c, expectedImage, expectedResources, predicates, limits, logger)
	if err != nil {
		logger.Error(err, "Getting customer pods to restart failed")
		return nil, false, err
	}

	warnings, err := restart.Restart(ctx, c, podsToRestart, logger, false)
	if err != nil {
		logger.Error(err, "Restarting customer pods failed")
		return nil, false, err
	}

	// if there are more pods to restart there should be a continue token in the pod list
	hasMorePodsToRestart := podsToRestart.Continue != ""

	if !hasMorePodsToRestart {
		logger.Info("Customer proxy reset completed")
	} else {
		logger.Info("Customer proxy reset only partially completed")
	}

	return warnings, hasMorePodsToRestart, nil
}
