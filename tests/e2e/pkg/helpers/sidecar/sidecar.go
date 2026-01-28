package sidecar

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

func VerifyIfPodHasIstioSidecar(pod *v1.Pod) error {
	for _, container := range pod.Spec.Containers {
		if container.Name == "istio-proxy" {
			return nil
		}
	}
	for _, container := range pod.Spec.InitContainers {
		if container.Name == "istio-proxy" {
			return nil
		}
	}

	return fmt.Errorf("sidecar 'istio-proxy' not found in pod %s", pod.Name)
}
