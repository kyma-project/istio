package v2

import (
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
)

// CompatibilityModeChanged is an Evaluator that detects a change in the Istio
// compatibility mode between the last applied config and the current CR.
type CompatibilityModeChanged struct {
	oldCompatibilityMode bool
	newCompatibilityMode bool
	hasProxyMetadata     bool
}

// Evaluate returns Restart when compatibility mode changed and proxy metadata is
// present, and Continue otherwise.
func (c *CompatibilityModeChanged) Evaluate(_ Object) Decision {
	// TODO: this logic will probably have to change once we move to actual workload evaluation
	//  we have to evaluate Object for actual state instead of validating the state inside the struct
	//  probably we have to used some kind of hashed state
	if c.hasProxyMetadata && c.oldCompatibilityMode != c.newCompatibilityMode {
		return Restart
	}
	return Continue
}

// NewCompatibilityModeChanged returns an Evaluator that restarts workloads when
// compatibility mode changed and the proxy metadata config is non-empty.
func NewCompatibilityModeChanged(cr operatorv1alpha2.Istio, lastApplied configuration.AppliedConfig, proxyMetaConfig map[string]string) Evaluator {
	return &CompatibilityModeChanged{
		oldCompatibilityMode: lastApplied.CompatibilityMode,
		newCompatibilityMode: cr.Spec.CompatibilityMode,
		hasProxyMetadata:     len(proxyMetaConfig) > 0,
	}
}
