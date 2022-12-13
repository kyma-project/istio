package pods

import (
	"encoding/json"
	v1 "k8s.io/api/core/v1"
)

const AnnotationResetWarningKey = "istio.reconciler.kyma-project.io/proxy-reset-warning"

func hasIstioSidecarStatusAnnotation(pod v1.Pod) bool {
	_, exists := pod.Annotations["sidecar.istio.io/status"]
	return exists
}

func isPodReady(pod v1.Pod) bool {
	isMarkedForDeletion := pod.ObjectMeta.DeletionTimestamp != nil
	return !isMarkedForDeletion && hasTrueStatusConditions(pod) && isPodRunning(pod)
}

func hasTrueStatusConditions(pod v1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Status != v1.ConditionTrue {
			return false
		}
	}
	return true
}

func isPodRunning(pod v1.Pod) bool {
	return pod.Status.Phase == v1.PodRunning
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

func hasIstioSidecarContainer(containers []v1.Container, istioSidecarName string) bool {
	for _, container := range containers {
		if isContainerIstioSidecar(container, []string{istioSidecarName}) {
			return true
		}
	}
	return false
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

func isPodInNamespaceList(pod v1.Pod, namespaceList []v1.Namespace) bool {
	for _, namespace := range namespaceList {
		if pod.ObjectMeta.Namespace == namespace.Name {
			return true
		}
	}
	return false
}

func isSystemNamespace(name string) bool {
	switch name {
	case "kube-system":
		return true
	case "kube-public":
		return true
	case "istio-system":
		return true
	}
	return false
}

func isPodEligibleToRestart(pod v1.Pod, isSidecarInjectionEnabledByDefault, podNamespaceLabeled bool) bool {
	podAnnotationValue, podAnnotated := pod.Annotations["sidecar.istio.io/inject"]
	podLabelValue, podLabeled := pod.Labels["sidecar.istio.io/inject"]

	if podLabeled && podLabelValue == "false" {
		return false
	}
	if !podLabeled && podAnnotated && podAnnotationValue == "false" {
		return false
	}
	if !isSidecarInjectionEnabledByDefault && !podNamespaceLabeled && podAnnotated && podAnnotationValue == "true" {
		return false
	}
	if !isSidecarInjectionEnabledByDefault && !podNamespaceLabeled && !podAnnotated && !podLabeled {
		return false
	}
	return true
}

func isPodInHostNetwork(pod v1.Pod) bool {
	//Automatic sidecar injection is ignored for pods on the host network
	return pod.Spec.HostNetwork
}
