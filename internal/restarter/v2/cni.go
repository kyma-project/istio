package v2

import corev1 "k8s.io/api/core/v1"

const (
	istioInitContainerName       = "istio-init"
	istioValidationContainerName = "istio-validation"
)

// CNIConfigChanged is an Evaluator that detects a change in the Istio CNI setting
// by inspecting a pod's init containers.
type CNIConfigChanged struct {
	disableCNI bool
}

// Evaluate returns Restart when the pod's init containers no longer match the
// expected CNI setting, and Continue otherwise.
func (c *CNIConfigChanged) Evaluate(obj Object) Decision {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return Continue
	}
	for _, container := range pod.Spec.InitContainers {
		if !c.disableCNI && container.Name == istioInitContainerName {
			return Restart
		}
		if c.disableCNI && container.Name == istioValidationContainerName {
			return Restart
		}
	}
	return Continue
}

// NewCNIChangedRule returns an Evaluator that restarts workloads whose init
// containers no longer match the given CNI setting.
func NewCNIChangedRule(disabled bool) Evaluator {
	return &CNIConfigChanged{disableCNI: disabled}
}
