package predicates

import (
	v1 "k8s.io/api/core/v1"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

type EnableDNSProxyingRestartPredicate struct {
	oldEnableDNSProxying *bool
	newEnableDNSProxying *bool
}

func NewEnableDNSProxyingRestartPredicate(istioCR *v1alpha2.Istio) (*EnableDNSProxyingRestartPredicate, error) {
	lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return nil, err
	}

	return &EnableDNSProxyingRestartPredicate{
		oldEnableDNSProxying: lastAppliedConfig.Config.EnableDNSProxying,
		newEnableDNSProxying: istioCR.Spec.Config.EnableDNSProxying,
	}, nil
}

func (p EnableDNSProxyingRestartPredicate) Matches(_ v1.Pod) bool {

	isNewNotNil := p.newEnableDNSProxying != nil
	isOldNotNil := p.oldEnableDNSProxying != nil
	if isNewNotNil && isOldNotNil && *p.oldEnableDNSProxying != *p.newEnableDNSProxying {
		return true
	} else if isNewNotNil != isOldNotNil {
		return true
	}

	return false
}

func (p EnableDNSProxyingRestartPredicate) MustMatch() bool {
	return false
}

func (p EnableDNSProxyingRestartPredicate) Name() string {
	return "EnableDNSProxyingRestartPredicate"
}
