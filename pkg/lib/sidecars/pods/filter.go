package pods

import (
	v1 "k8s.io/api/core/v1"
)

const (
	istioSidecarName = "istio-proxy"
)

func needsRestart(pod v1.Pod, expectedImage SidecarImage, expectedResources v1.ResourceRequirements) bool {
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

func hasDifferentSidecarResources(pod v1.Pod, expectedResources v1.ResourceRequirements) bool {

	for _, container := range pod.Spec.Containers {
		if isContainerIstioSidecar(container) && !containerHasResources(container, expectedResources) {
			return true
		}
	}
	return false
}

func containerHasResources(container v1.Container, expectedResources v1.ResourceRequirements) bool {
	equalCpuRequests := container.Resources.Requests.Cpu().Equal(*expectedResources.Requests.Cpu())
	equalMemoryRequests := container.Resources.Requests.Memory().Equal(*expectedResources.Requests.Memory())
	equalCpuLimits := container.Resources.Limits.Cpu().Equal(*expectedResources.Limits.Cpu())
	equalMemoryLimits := container.Resources.Limits.Memory().Equal(*expectedResources.Limits.Memory())

	return equalCpuRequests && equalMemoryRequests && equalCpuLimits && equalMemoryLimits
}

func isContainerIstioSidecar(container v1.Container) bool {
	return istioSidecarName == container.Name
}
