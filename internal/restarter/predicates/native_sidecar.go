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
	isNativeSidecar := false
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == "istio-proxy" {
			isNativeSidecar = true
			break
		}
	}

	if !isNativeSidecar && !p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] == "" {
		return true
	}

	if !isNativeSidecar && !p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] == "true" {
		return true
	}

	if !isNativeSidecar && p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] == "true" {
		return true
	}

	if isNativeSidecar && p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] == "" {
		return true
	}

	if isNativeSidecar && !p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] == "false" {
		return true
	}

	if isNativeSidecar && p.compatibilityMode && pod.Annotations[nativeSidecarAnnotation] == "false" {
		return true
	}

	return false
}

func (p *NativeSidecarRestartPredicate) MustMatch() bool {
	return false
}
