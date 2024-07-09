package compatibility

import (
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	v1 "k8s.io/api/core/v1"
)

type ProxyRestartPredicate struct {
	oldCompatibilityMode bool
	newCompatibilityMode bool
	config               config
}

func NewRestartPredicate(istioCR *v1alpha2.Istio) (*ProxyRestartPredicate, error) {
	lastAppliedConfig, err := istio.GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return nil, err
	}

	return &ProxyRestartPredicate{
		oldCompatibilityMode: lastAppliedConfig.IstioSpec.CompatibilityMode,
		newCompatibilityMode: istioCR.Spec.CompatibilityMode,
		config:               config{proxyMetadata: v1alpha2.ProxyMetaDataCompatibility},
	}, nil
}

type config struct {
	proxyMetadata map[string]string
}

func (c config) hasProxyMetadata() bool {
	return len(c.proxyMetadata) > 0
}

func (p ProxyRestartPredicate) RequiresProxyRestart(_ v1.Pod) bool {
	if p.config.hasProxyMetadata() && p.oldCompatibilityMode != p.newCompatibilityMode {
		return true
	}

	return false
}
