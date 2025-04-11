package predicates

import (
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	v1 "k8s.io/api/core/v1"
)

type CompatibilityRestartPredicate struct {
	oldCompatibilityMode bool
	newCompatibilityMode bool
	config               config
}

func NewCompatibilityRestartPredicate(istioCR *v1alpha2.Istio) (*CompatibilityRestartPredicate, error) {
	lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return nil, err
	}

	return &CompatibilityRestartPredicate{
		oldCompatibilityMode: lastAppliedConfig.CompatibilityMode,
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

func (p CompatibilityRestartPredicate) Matches(_ v1.Pod) bool {
	if p.config.hasProxyMetadata() && p.oldCompatibilityMode != p.newCompatibilityMode {
		return true
	}

	return false
}

func (p CompatibilityRestartPredicate) MustMatch() bool {
	return false
}
