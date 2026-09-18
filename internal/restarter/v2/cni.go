package v2

import corev1 "k8s.io/api/core/v1"

const (
	istioInitContainerName       = "istio-init"
	istioValidationContainerName = "istio-validation"
)

type CNIConfigChanged struct {
	disableCNI bool
}

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

func NewCNIChangedRule(disabled bool) Rule {
	return &CNIConfigChanged{disableCNI: disabled}
}
