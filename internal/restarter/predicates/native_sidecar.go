package predicates

import (
	v1 "k8s.io/api/core/v1"
)

const nativeSidecarAnnotation = "sidecar.istio.io/nativeSidecar"

type NativeSidecarRestartPredicate struct{}

func NewNativeSidecarRestartPredicate() *NativeSidecarRestartPredicate {
	return &NativeSidecarRestartPredicate{}
}

func (p *NativeSidecarRestartPredicate) Matches(pod v1.Pod) bool {
	isPodWithNativeSidecar := false
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == "istio-proxy" {
			isPodWithNativeSidecar = true
			break
		}
	}

	return (pod.Annotations[nativeSidecarAnnotation] != "false") != isPodWithNativeSidecar
}

func (p *NativeSidecarRestartPredicate) MustMatch() bool {
	return false
}

func (p *NativeSidecarRestartPredicate) Name() string {
	return "NativeSidecarRestartPredicate"
}
