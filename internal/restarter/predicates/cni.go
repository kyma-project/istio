package predicates

import (
	v1 "k8s.io/api/core/v1"
)

const (
	istioInitContainerName       = "istio-init"
	istioValidationContainerName = "istio-validation"
)

// CniRestartPredicate restarts pods that have the wrong init container for the current CNI mode.
// When CNI is enabled (disableCni: false), pods should use istio-validation instead of istio-init.
// When CNI is disabled (disableCni: true), pods should use istio-init instead of istio-validation.
type CniRestartPredicate struct {
	disableCni bool
}

func NewCniRestartPredicate(disableCni bool) *CniRestartPredicate {
	return &CniRestartPredicate{
		disableCni: disableCni,
	}
}

// Matches returns true when the pod has a stale init container for the current CNI mode:
//   - disableCni: false → CNI is ON,  pods must use istio-validation; restart those that still have istio-init
//   - disableCni: true  → CNI is OFF, pods must use istio-init;       restart those that still have istio-validation
func (p CniRestartPredicate) Matches(pod v1.Pod) bool {
	for _, container := range pod.Spec.InitContainers {
		if !p.disableCni && container.Name == istioInitContainerName {
			return true
		}
		if p.disableCni && container.Name == istioValidationContainerName {
			return true
		}
	}
	return false
}

func (p CniRestartPredicate) MustMatch() bool {
	return false
}

func (p CniRestartPredicate) Name() string {
	return "CniRestartPredicate"
}
