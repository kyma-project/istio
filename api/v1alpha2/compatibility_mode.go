package v1alpha2

import (
	"istio.io/api/operator/v1alpha1"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

// the following map contains Istio compatibility environment variables, that are not included in the compatibilityVersion of istioctl install
// should be updated with every Istio bump according to the release notes
// current env comes from: Istio 1.21, compatibilityVersion 1.20
var pilotCompatibilityEnvVars = map[string]string{
	"PERSIST_OLDEST_FIRST_HEURISTIC_FOR_VIRTUAL_SERVICE_HOST_MATCHING": "true",
	"VERIFY_CERTIFICATE_AT_CLIENT":                                     "false",
	"ENABLE_AUTO_SNI":                                                  "false",
}

func setCompatibilityMode(op iopv1alpha1.IstioOperator) iopv1alpha1.IstioOperator {
	pilotIop := setCompatibilityPilot(op)
	return pilotIop
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
