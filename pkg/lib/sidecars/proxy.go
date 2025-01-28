package sidecars

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

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
	k8sClient       client.Client
	podsLister      pods.PodsGetter
	actionRestarter restart.ActionRestarter
	logger          *logr.Logger
}

func NewProxyRestarter(c client.Client, podsLister pods.PodsGetter, actionRestarter restart.ActionRestarter, logger *logr.Logger) *ProxyRestart {
	return &ProxyRestart{
		k8sClient:       c,
		podsLister:      podsLister,
		actionRestarter: actionRestarter,
		logger:          logger,
	}
}

func (p *ProxyRestart) RestartProxies(ctx context.Context, expectedImage predicates.SidecarImage, expectedResources v1.ResourceRequirements, istioCR *v1alpha2.Istio) ([]restart.RestartWarning, bool, error) {
	compatibiltyPredicate, err := predicates.NewCompatibilityRestartPredicate(istioCR)
	if err != nil {
		p.logger.Error(err, "Failed to create restart compatibility predicate")
		return []restart.RestartWarning{}, false, err
	}

	predicates := []predicates.SidecarProxyPredicate{compatibiltyPredicate,
		predicates.NewImageResourcesPredicate(expectedImage, expectedResources),
	}

	err = p.restartKymaProxies(ctx, predicates)
	if err != nil {
		p.logger.Error(err, "Failed to restart Kyma proxies")
		return []restart.RestartWarning{}, false, err
	}

	warnings, hasMorePodsToRestart, err := p.restartCustomerProxies(ctx, predicates)
	if err != nil {
		p.logger.Error(err, "failed to restart Customer proxies")
		warnings = append(warnings, restart.RestartWarning{
			Name:      "n/a",
			Namespace: "n/a",
			Kind:      "n/a",
			Message:   "failed to restart Customer proxies",
		})
	}

	return warnings, hasMorePodsToRestart, nil
}

func (p *ProxyRestart) RestartWithPredicates(ctx context.Context, preds []predicates.SidecarProxyPredicate, limits *pods.PodsRestartLimits, failOnError bool) ([]restart.RestartWarning, bool, error) {
	podsToRestart, err := p.podsLister.GetPodsToRestart(ctx, preds, limits)
	if err != nil {
		p.logger.Error(err, "Getting pods to restart failed")
		return []restart.RestartWarning{}, false, err
	}

	warnings, err := p.actionRestarter.RestartAction(ctx, podsToRestart, failOnError)
	if err != nil {
		p.logger.Error(err, "Restarting pods failed")
		return warnings, false, err
	}

	// if there are more pods to restart there should be a continue token in the pod list
	return warnings, podsToRestart.Continue != "", nil
}

func (p *ProxyRestart) restartKymaProxies(ctx context.Context, preds []predicates.SidecarProxyPredicate) error {
	preds = append(preds, predicates.NewKymaWorkloadRestartPredicate())
	limits := pods.NewPodsRestartLimits(math.MaxInt, math.MaxInt)

	warnings, _, err := p.RestartWithPredicates(ctx, preds, limits, true)
	if err != nil {
		p.logger.Error(err, "Failed to restart Kyma proxies")
		return err
	}
	warningMessage := BuildWarningMessage(warnings, p.logger)
	if warningMessage != "" {
		err := errors.New(warningMessage)
		p.logger.Error(err, "Failed to restart Kyma proxies")
		return err
	}

	p.logger.Info("Kyma proxy restart completed")
	return nil
}

func BuildWarningMessage(warnings []restart.RestartWarning, logger *logr.Logger) string {
	warningMessage := ""
	warningsCount := len(warnings)
	if warningsCount > 0 {
		podsLimit := 5
		pods := []string{}
		for _, w := range warnings {
			if podsLimit--; podsLimit >= 0 {
				pods = append(pods, fmt.Sprintf("%s/%s", w.Namespace, w.Name))
			}
			logger.Info("Proxy reset failed:", "name", w.Name, "namespace", w.Namespace, "kind", w.Kind, "message", w.Message)
		}
		warningMessage = fmt.Sprintf("The sidecars of the following workloads could not be restarted: %s",
			strings.Join(pods, ", "))
		if warningsCount-len(pods) > 0 {
			warningMessage += fmt.Sprintf(" and %d additional workload(s)", warningsCount-len(pods))
		}
	}
	return warningMessage
}

func (p *ProxyRestart) restartCustomerProxies(ctx context.Context, preds []predicates.SidecarProxyPredicate) ([]restart.RestartWarning, bool, error) {
	preds = append(preds, predicates.NewCustomerWorkloadRestartPredicate())
	limits := pods.NewPodsRestartLimits(podsToRestartLimit, podsToListLimit)

	warnings, hasMorePodsToRestart, err := p.RestartWithPredicates(ctx, preds, limits, false)
	if err != nil {
		p.logger.Error(err, "Failed to restart Customer proxies")
		return warnings, false, err
	}

	if !hasMorePodsToRestart {
		p.logger.Info("Customer proxy restart completed")
	} else {
		p.logger.Info("Customer proxy restart only partially completed")
	}

	return warnings, hasMorePodsToRestart, nil
}
