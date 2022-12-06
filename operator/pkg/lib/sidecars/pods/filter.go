package pods

import (
	"encoding/json"
	v1 "k8s.io/api/core/v1"
)

func hasIstioSidecarStatusAnnotation(pod v1.Pod) bool {
	_, exists := pod.Annotations["sidecar.istio.io/status"]
	return exists
}

func isPodReady(pod v1.Pod) bool {
	isMarkedForDeletion := pod.ObjectMeta.DeletionTimestamp != nil
	return !isMarkedForDeletion && hasTrueStatusConditions(pod)
}

func hasTrueStatusConditions(pod v1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Status != v1.ConditionTrue {
			return false
		}
	}

	return true
}

func getIstioSidecarNamesInPod(pod v1.Pod) []string {
	type istioStatusStruct struct {
		Containers []string `json:"containers"`
	}
	istioStatus := istioStatusStruct{}
	err := json.Unmarshal([]byte(pod.Annotations["sidecar.istio.io/status"]), &istioStatus)
	if err != nil {
		return []string{}
	}
	return istioStatus.Containers
}

func hasSidecarContainerWithWithDifferentImage(pod v1.Pod, expectedImage SidecarImage) bool {
	// TODO Why can't we simply assume the pod name to be istio-proxy? Because of istio-init?
	istioContainerNames := getIstioSidecarNamesInPod(pod)

	for _, container := range pod.Spec.Containers {
		if isContainerIstioSidecar(container, istioContainerNames) && !expectedImage.matchesImageIn(container) {
			return true
		}
	}
	return false
}

func hasInitContainer(containers []v1.Container, initContainerName string) bool {
	proxyImage := ""
	for _, container := range containers {
		if container.Name == initContainerName {
			proxyImage = container.Image
		}
	}
	return proxyImage != ""
}

func isContainerIstioSidecar(container v1.Container, istioSidecarNames []string) bool {
	for _, sidecarName := range istioSidecarNames {
		if sidecarName == container.Name {
			return true
		}
	}
	return false
}

func HasResetWarning(pod v1.Pod) bool {
	_, exists := pod.Annotations[AnnotationResetWarningKey]
	return exists
}
