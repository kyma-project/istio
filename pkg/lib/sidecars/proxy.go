package sidecars

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/images"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
)

const (
	podsLimitToRestartPerPage = 30
)

type ProxyRestarter interface {
	RestartProxies(
		ctx context.Context,
		expectedImage images.Image,
		expectedResources v1.ResourceRequirements,
		istioCR *v1alpha2.Istio,
	) ([]restart.Warning, error)
	RestartWithPredicates(ctx context.Context, preds []predicates.SidecarProxyPredicate, limits *pods.RestartLimits, failOnError bool) ([]restart.Warning, error)
}

type ProxyRestart struct {
	k8sClient       client.Client
	podsLister      pods.Getter
	actionRestarter restart.ActionRestarter
	logger          *logr.Logger
}

func NewProxyRestarter(c client.Client, podsLister pods.Getter, actionRestarter restart.ActionRestarter, logger *logr.Logger) *ProxyRestart {
	return &ProxyRestart{
		k8sClient:       c,
		podsLister:      podsLister,
		actionRestarter: actionRestarter,
		logger:          logger,
	}
}

func (p *ProxyRestart) RestartProxies(
	ctx context.Context,
	expectedImage images.Image,
	expectedResources v1.ResourceRequirements,
	istioCR *v1alpha2.Istio,
) ([]restart.Warning, error) {
	compatibiltyPredicate, err := predicates.NewCompatibilityRestartPredicate(istioCR)
	if err != nil {
		p.logger.Error(err, "Failed to create restart compatibility predicate")
		return []restart.Warning{}, err
	}
	prometheusMergePredicate, err := predicates.NewPrometheusMergeRestartPredicate(ctx, p.k8sClient, istioCR)
	if err != nil {
		p.logger.Error(err, "Failed to create restart prometheusMerge predicate")
		return []restart.Warning{}, err
	}
	enableDNSProxyingPredicate, err := predicates.NewEnableDNSProxyingRestartPredicate(istioCR)
	if err != nil {
		p.logger.Error(err, "Failed to create restart enableDNSProxying predicate")
		return []restart.Warning{}, err
	}
	predicates := []predicates.SidecarProxyPredicate{
		compatibiltyPredicate,
		prometheusMergePredicate,
		predicates.NewImageResourcesPredicate(expectedImage, expectedResources),
		predicates.NewNativeSidecarRestartPredicate(istioCR),
		enableDNSProxyingPredicate,
	}

	err = p.restartKymaProxies(ctx, predicates)
	if err != nil {
		p.logger.Error(err, "Failed to restart Kyma proxies")
		return []restart.Warning{}, err
	}

	warnings, err := p.restartCustomerProxies(ctx, predicates)
	if err != nil {
		p.logger.Error(err, "failed to restart Customer proxies")
		warnings = []restart.Warning{ // errors on Customer proxies are considered as a warning
			{
				Name:      "n/a",
				Namespace: "n/a",
				Kind:      "n/a",
				Message:   "failed to restart Customer proxies",
			},
		}
	}

	return warnings, nil
}

func (p *ProxyRestart) RestartWithPredicates(
	ctx context.Context,
	preds []predicates.SidecarProxyPredicate,
	limits *pods.RestartLimits,
	failOnError bool,
) ([]restart.Warning, error) {
	var allWarnings []restart.Warning

	err := p.podsLister.GetPodsToRestart(ctx, preds, limits, func(ctx context.Context, page *v1.PodList) error {
		warnings, err := p.actionRestarter.Restart(ctx, page, failOnError)
		allWarnings = append(allWarnings, warnings...)
		if err != nil {
			p.logger.Error(err, "Restarting pods failed")
			return err
		}
		return nil
	})
	if err != nil {
		p.logger.Error(err, "Getting pods to restart failed")
		return allWarnings, err
	}

	return allWarnings, nil
}

func (p *ProxyRestart) restartKymaProxies(ctx context.Context, preds []predicates.SidecarProxyPredicate) error {
	preds = append(preds, predicates.NewKymaWorkloadRestartPredicate())
	limits := pods.NewPodsRestartLimits(podsLimitToRestartPerPage)

	warnings, err := p.RestartWithPredicates(ctx, preds, limits, true)
	if err != nil {
		p.logger.Error(err, "Failed to restart Kyma proxies")
		return err
	}
	warningMessage := BuildWarningMessage(warnings, p.logger)
	if warningMessage != "" {
		err = errors.New(warningMessage)
		p.logger.Error(err, "Failed to restart Kyma proxies")
		return err
	}

	p.logger.Info("Kyma proxy restart completed")
	return nil
}

func BuildWarningMessage(warnings []restart.Warning, logger *logr.Logger) string {
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

func (p *ProxyRestart) restartCustomerProxies(ctx context.Context, preds []predicates.SidecarProxyPredicate) ([]restart.Warning, error) {
	preds = append(preds, predicates.NewCustomerWorkloadRestartPredicate())
	limits := pods.NewPodsRestartLimits(podsLimitToRestartPerPage)

	warnings, err := p.RestartWithPredicates(ctx, preds, limits, false)
	if err != nil {
		p.logger.Error(err, "Failed to restart Customer proxies")
		return warnings, err
	}

	p.logger.Info("Customer proxy restart completed")

	return warnings, nil
}
