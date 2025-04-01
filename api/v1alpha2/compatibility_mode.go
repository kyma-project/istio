package v1alpha2

import (
	iopv1alpha1 "istio.io/istio/operator/pkg/apis"
	v1 "k8s.io/api/core/v1"
)

var pilotCompatibilityEnvVars = map[string]string{
	"ENABLE_INBOUND_RETRY_POLICY":                      "false",
	"EXCLUDE_UNSAFE_503_FROM_DEFAULT_RETRY":            "false",
	"PREFER_DESTINATIONRULE_TLS_FOR_EXTERNAL_SERVICES": "false",
	"ENABLE_ENHANCED_DESTINATIONRULE_MERGE":            "false",
	"PILOT_UNIFIED_SIDECAR_SCOPE":                      "false",
	"PILOT_ENABLE_IP_AUTOALLOCATE":                     "false",
}

func setCompatibilityMode(op iopv1alpha1.IstioOperator) (iopv1alpha1.IstioOperator, error) {
	pilotIop := setCompatibilityPilot(op)
	return setCompatibilityProxyMetadata(pilotIop)
}

func setCompatibilityPilot(op iopv1alpha1.IstioOperator) iopv1alpha1.IstioOperator {
	if op.Spec.Components == nil {
		op.Spec.Components = &iopv1alpha1.IstioComponentSpec{}
	}
	if op.Spec.Components.Pilot == nil {
		op.Spec.Components.Pilot = &iopv1alpha1.ComponentSpec{}
	}
	if op.Spec.Components.Pilot.Kubernetes == nil {
		op.Spec.Components.Pilot.Kubernetes = &iopv1alpha1.KubernetesResources{}
	}

	for k, v := range pilotCompatibilityEnvVars {
		op.Spec.Components.Pilot.Kubernetes.Env = append(op.Spec.Components.Pilot.Kubernetes.Env, &v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	return op
}

var ProxyMetaDataCompatibility = map[string]string{
	"ENABLE_DEFERRED_STATS_CREATION":               "false",
	"BYPASS_OVERLOAD_MANAGER_FOR_STATIC_LISTENERS": "false",
}

func setCompatibilityProxyMetadata(op iopv1alpha1.IstioOperator) (iopv1alpha1.IstioOperator, error) {
	mcb, err := newMeshConfigBuilder(op)
	if err != nil {
		return op, err
	}

	for k, v := range ProxyMetaDataCompatibility {
		mcb, err = mcb.AddProxyMetadata(k, v)
		if err != nil {
			return iopv1alpha1.IstioOperator{}, err
		}
	}

	op.Spec.MeshConfig = mcb.Build()

	return op, nil
}
