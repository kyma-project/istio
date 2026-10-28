package v2

import (
	"errors"
	"fmt"

	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/istiofeatures"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Config struct {
	IstioCR           *istioCR.Istio
	LastAppliedConfig configuration.AppliedConfig
	BetaFeatures      istiofeatures.IstioFeatures
	// SidecarResources is the expected Istio proxy resource spec; a pod whose sidecar differs is restarted.
	SidecarResources corev1.ResourceRequirements
}

func GetEvaluators(cfg Config) (evaluators []Evaluator, err error) {
	if cfg.IstioCR == nil {
		return nil, errors.New("istio CR is nil")
	}
	sidecarResources, err := effectiveSidecarResources(cfg.LastAppliedConfig.Components, cfg.SidecarResources)
	if err != nil {
		return nil, err
	}
	evaluators = append(evaluators, NewCNIChangedRule(cfg.BetaFeatures.DisableCni))
	evaluators = append(evaluators, NewCompatibilityModeChanged(*cfg.IstioCR, cfg.LastAppliedConfig, istioCR.ProxyMetaDataCompatibility))
	evaluators = append(evaluators, NewDNSProxyingChanged(*cfg.IstioCR, cfg.LastAppliedConfig))
	evaluators = append(evaluators, NewNumTrustedProxiesConfigChanged(*cfg.IstioCR, cfg.LastAppliedConfig))
	evaluators = append(evaluators, NewForwardClientCertDetailsConfigChanged(*cfg.IstioCR, cfg.LastAppliedConfig))
	evaluators = append(evaluators, NewTrustDomainConfigChanged(*cfg.IstioCR, cfg.LastAppliedConfig))
	evaluators = append(evaluators, NewProxyStatsMatcherConfigChanged(cfg.IstioCR, cfg.LastAppliedConfig))
	evaluators = append(evaluators, NewPrometheusMergeConfigChanged(
		cfg.IstioCR.Spec.Config.Telemetry.Metrics.PrometheusMerge,
		15020,
	))
	evaluators = append(evaluators, NewSidecarImageChangedRule(cfg.LastAppliedConfig.IstioTag))
	evaluators = append(evaluators, NewSidecarResourcesChangedRule(sidecarResources))

	return evaluators, nil
}

// effectiveSidecarResources overlays the proxy resources recorded in the last
// applied configuration (components.proxy.k8s.resources) onto base, overriding
// only the individual CPU/memory request and limit fields that config actually
// sets. This mirrors how the operator merges proxy resources over the operator
// defaults when rendering the sidecar, so the restart evaluator compares against
// the same expected values the last applied config produced.
func effectiveSidecarResources(components *istioCR.Components, base corev1.ResourceRequirements) (corev1.ResourceRequirements, error) {
	result := base.DeepCopy()

	if components == nil || components.Proxy == nil ||
		components.Proxy.K8S == nil || components.Proxy.K8S.Resources == nil {
		return *result, nil
	}
	res := components.Proxy.K8S.Resources

	if result.Limits == nil {
		result.Limits = corev1.ResourceList{}
	}
	if result.Requests == nil {
		result.Requests = corev1.ResourceList{}
	}

	if res.Limits != nil {
		if err := overrideQuantity(result.Limits, corev1.ResourceCPU, res.Limits.CPU); err != nil {
			return corev1.ResourceRequirements{}, err
		}
		if err := overrideQuantity(result.Limits, corev1.ResourceMemory, res.Limits.Memory); err != nil {
			return corev1.ResourceRequirements{}, err
		}
	}
	if res.Requests != nil {
		if err := overrideQuantity(result.Requests, corev1.ResourceCPU, res.Requests.CPU); err != nil {
			return corev1.ResourceRequirements{}, err
		}
		if err := overrideQuantity(result.Requests, corev1.ResourceMemory, res.Requests.Memory); err != nil {
			return corev1.ResourceRequirements{}, err
		}
	}

	return *result, nil
}

// overrideQuantity sets list[name] to the parsed value when value is non-nil.
// list must be non-nil. A nil value leaves the entry untouched.
func overrideQuantity(list corev1.ResourceList, name corev1.ResourceName, value *string) error {
	if value == nil {
		return nil
	}
	q, err := resource.ParseQuantity(*value)
	if err != nil {
		return fmt.Errorf("parse proxy resource %s %q: %w", name, *value, err)
	}
	list[name] = q
	return nil
}
