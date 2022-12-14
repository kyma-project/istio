package pods

import (
	v1 "k8s.io/api/core/v1"
)

const (
	istioSidecarName = "istio-proxy"
)

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

func hasSidecarContainerWithWithDifferentImage(pod v1.Pod, expectedImage SidecarImage) bool {

	for _, container := range pod.Spec.Containers {
		if isContainerIstioSidecar(container) && !expectedImage.matchesImageIn(container) {
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

func hasIstioSidecarContainer(containers []v1.Container) bool {
	for _, container := range containers {
		if isContainerIstioSidecar(container) {
			return true
		}
	}
	return false
}

func isContainerIstioSidecar(container v1.Container) bool {
	return istioSidecarName == container.Name
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
