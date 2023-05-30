package pods

import (
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
)

const (
	istioSidecarName = "istio-proxy"
)

func needsRestart(pod v1.Pod, expectedImage SidecarImage, expectedResources v1alpha1.Resources) bool {
	return hasIstioSidecarStatusAnnotation(pod) &&
		isPodReady(pod) &&
		(hasSidecarContainerWithWithDifferentImage(pod, expectedImage) || hasDifferentSidecarResources(pod, expectedResources))
}

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

func hasDifferentSidecarResources(pod v1.Pod, expectedResources v1alpha1.Resources) bool {

	for _, container := range pod.Spec.Containers {
		if isContainerIstioSidecar(container) && !expectedResources.IsEqual(container.Resources) {
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
