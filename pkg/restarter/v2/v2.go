package v2

import (
	"errors"

	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/istiofeatures"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	corev1 "k8s.io/api/core/v1"
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
	evaluators = append(evaluators, NewSidecarResourcesChangedRule(cfg.SidecarResources))

	return evaluators, nil
}
