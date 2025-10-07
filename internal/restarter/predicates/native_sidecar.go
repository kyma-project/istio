package predicates

import (
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	v1 "k8s.io/api/core/v1"
)

const nativeSidecarAnnotation = "sidecar.istio.io/nativeSidecar"

type NativeSidecarRestartPredicate struct {
	compatibilityMode bool
}

func NewNativeSidecarRestartPredicate(istioCR *v1alpha2.Istio) *NativeSidecarRestartPredicate {
	compatibilityMode := istioCR.Spec.CompatibilityMode
	return &NativeSidecarRestartPredicate{
		compatibilityMode,
	}
}

func (p *NativeSidecarRestartPredicate) Matches(pod v1.Pod) bool {
	isPodWithNativeSidecar := false
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == "istio-proxy" {
			isPodWithNativeSidecar = true
			break
		}
	}
	if isPodWithNativeSidecar {
		if p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] != "true" {
			return true
		}
		if !p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] == "false" {
			return true
		}
	} else {
		if !p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] != "false" {
			return true
		}
		if p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] == "true" {
			return true
		}
	}

	return false
}

func (p *NativeSidecarRestartPredicate) MustMatch() bool {
	return false
}
