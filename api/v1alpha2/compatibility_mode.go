package v1alpha2

import (
	"istio.io/api/operator/v1alpha1"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

var pilotCompatibilityEnvVars = map[string]string{
	"ENABLE_DELIMITED_STATS_TAG_REGEX": "false",
}

func setCompatibilityMode(op iopv1alpha1.IstioOperator) (iopv1alpha1.IstioOperator, error) {
	pilotIop := setCompatibilityPilot(op)
	return setCompatibilityProxyMetadata(pilotIop)
}

func setCompatibilityPilot(op iopv1alpha1.IstioOperator) iopv1alpha1.IstioOperator {
	if op.Spec == nil {
		op.Spec = &v1alpha1.IstioOperatorSpec{}
	}
	if op.Spec.Components == nil {
		op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
	}
	if op.Spec.Components.Pilot == nil {
		op.Spec.Components.Pilot = &v1alpha1.ComponentSpec{}
	}
	if op.Spec.Components.Pilot.K8S == nil {
		op.Spec.Components.Pilot.K8S = &v1alpha1.KubernetesResourcesSpec{}
	}

	for k, v := range pilotCompatibilityEnvVars {
		op.Spec.Components.Pilot.K8S.Env = append(op.Spec.Components.Pilot.K8S.Env, &v1alpha1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	return op
}

var ProxyMetaDataCompatibility = map[string]string{
	"ENABLE_DEFERRED_CLUSTER_CREATION": "false",
	"ENABLE_DELIMITED_STATS_TAG_REGEX": "false",
}

func setCompatibilityProxyMetadata(op iopv1alpha1.IstioOperator) (iopv1alpha1.IstioOperator, error) {
	if op.Spec == nil {
		op.Spec = &v1alpha1.IstioOperatorSpec{}
	}

	mcb, err := newMeshConfigBuilder(op)
	if err != nil {
		return op, err
	}

	for k, v := range ProxyMetaDataCompatibility {
		mcb.AddProxyMetadata(k, v)
	}
	newMeshConfig := mcb.Build()

	updatedConfig, err := marshalMeshConfig(newMeshConfig)
	if err != nil {
		return op, err
	}

	op.Spec.MeshConfig = updatedConfig

	return op, nil
}
