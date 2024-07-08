package pods

import (
	v1 "k8s.io/api/core/v1"
)

const (
	istioSidecarName                         = "istio-proxy"
	istioSidecarCustomImageAnnotation string = "sidecar.istio.io/proxyImage"
)

type RestartProxyPredicate struct {
	expectedImage     SidecarImage
	expectedResources v1.ResourceRequirements
}

func NewRestartProxyPredicate(expectedImage SidecarImage, expectedResources v1.ResourceRequirements) *RestartProxyPredicate {
	return &RestartProxyPredicate{expectedImage: expectedImage, expectedResources: expectedResources}
}

func (p RestartProxyPredicate) RequiresProxyRestart(pod v1.Pod) bool {
	return needsRestart(pod, p.expectedImage, *p.expectedResources.DeepCopy())
}

func needsRestart(pod v1.Pod, expectedImage SidecarImage, expectedResources v1.ResourceRequirements) bool {
	return HasIstioSidecarStatusAnnotation(pod) &&
		IsPodReady(pod) &&
		!hasCustomImageAnnotation(pod) &&
		(hasSidecarContainerWithWithDifferentImage(pod, expectedImage) || hasDifferentSidecarResources(pod, expectedResources))
}

func HasIstioSidecarStatusAnnotation(pod v1.Pod) bool {
	_, exists := pod.Annotations["sidecar.istio.io/status"]
	return exists
}

func IsPodReady(pod v1.Pod) bool {
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

func hasCustomImageAnnotation(pod v1.Pod) bool {
	_, found := pod.Annotations[istioSidecarCustomImageAnnotation]
	return found
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
