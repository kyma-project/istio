package v2

import (
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

// DNSProxyingChanged is an Evaluator that detects a change in the Istio DNS
// proxying setting between the last applied config and the current CR.
type DNSProxyingChanged struct {
	oldEnableDNSProxying *bool
	newEnableDNSProxying *bool
}

// Evaluate returns Restart when the DNS proxying setting changed, and Continue
// otherwise.
func (c *DNSProxyingChanged) Evaluate(_ Object) Decision {
	// TODO: this logic will probably have to change once we move to actual workload evaluation
	//  we have to evaluate Object for actual state instead of validating the state inside the struct
	//  probably we have to used some kind of hashed state
	if ptrEqual(c.oldEnableDNSProxying, c.newEnableDNSProxying) {
		return Continue
	}
	return Restart
}

// NewDNSProxyingChanged returns an Evaluator that restarts workloads when the
// DNS proxying setting changed between the last applied config and the CR.
func NewDNSProxyingChanged(cr operatorv1alpha2.Istio, lastApplied configuration.AppliedConfig) Evaluator {
	return &DNSProxyingChanged{
		oldEnableDNSProxying: lastApplied.Config.EnableDNSProxying,
		newEnableDNSProxying: cr.Spec.Config.EnableDNSProxying,
	}
}
