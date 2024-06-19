package compatibility

import (
	"context"
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	v1 "k8s.io/api/core/v1"
)

type ProxyRestartPredicate struct {
	istioCR *v1alpha2.Istio
	config  config
}

func NewRestartPredicate(istioCR *v1alpha2.Istio) *ProxyRestartPredicate {
	return &ProxyRestartPredicate{istioCR: istioCR, config: config{proxyMetadata: v1alpha2.ProxyMetaDataCompatibility}}
}

type config struct {
	proxyMetadata map[string]string
}

func (c config) hasProxyMetadata() bool {
	return len(c.proxyMetadata) > 0
}

func (p ProxyRestartPredicate) NewProxyRestartEvaluator(_ context.Context) (filter.ProxyRestartEvaluator, error) {
	lastAppliedConfig, err := istio.GetLastAppliedConfiguration(p.istioCR)
	if err != nil {
		return nil, err
	}

	return ProxiesRestartEvaluator{
		oldCompatibilityMode: lastAppliedConfig.IstioSpec.CompatibilityMode,
		newCompatibilityMode: p.istioCR.Spec.CompatibilityMode,
		config:               p.config,
	}, nil
}

type ProxiesRestartEvaluator struct {
	oldCompatibilityMode bool
	newCompatibilityMode bool
	config               config
}

func (p ProxiesRestartEvaluator) RequiresProxyRestart(_ v1.Pod) bool {

	if p.config.hasProxyMetadata() && p.oldCompatibilityMode != p.newCompatibilityMode {
		return true
	}

	return false
}
