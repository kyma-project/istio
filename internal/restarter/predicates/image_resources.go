package predicates

import (
	"fmt"
	"k8s.io/apimachinery/pkg/api/resource"

	v1 "k8s.io/api/core/v1"
)

const (
	istioSidecarName                         = "istio-proxy"
	istioSidecarCustomImageAnnotation string = "sidecar.istio.io/proxyImage"
)

type SidecarImage struct {
	Repository string
	Tag        string
}

func NewSidecarImage(hub, tag string) SidecarImage {
	return SidecarImage{
		Repository: fmt.Sprintf("%s/proxyv2", hub),
		Tag:        tag,
	}
}

func (r SidecarImage) String() string {
	return fmt.Sprintf("%s:%s", r.Repository, r.Tag)
}

func (r SidecarImage) matchesImageIn(container v1.Container) bool {
	return container.Image == r.String()
}

type ImageResourcesPredicate struct {
	expectedImage     SidecarImage
	expectedResources v1.ResourceRequirements
}

// NewImageResourcesPredicate creates a new ImageResourcesPredicate that checks if a pod needs a restart based on the expected image and resources.
func NewImageResourcesPredicate(expectedImage SidecarImage, expectedResources v1.ResourceRequirements) *ImageResourcesPredicate {
	return &ImageResourcesPredicate{expectedImage: expectedImage, expectedResources: expectedResources}
}

func (p ImageResourcesPredicate) Matches(pod v1.Pod) bool {
	return needsRestart(pod, p.expectedImage, *p.expectedResources.DeepCopy())
}

func (p ImageResourcesPredicate) MustMatch() bool {
	return false
}

func needsRestart(pod v1.Pod, expectedImage SidecarImage, expectedResources v1.ResourceRequirements) bool {
	return !hasCustomImageAnnotation(pod) &&
		(hasSidecarContainerWithWithDifferentImage(pod, expectedImage) || hasDifferentSidecarResources(pod, expectedResources))
}

func IsReadyWithIstioAnnotation(pod v1.Pod) bool {
	return IsPodReady(pod) && HasIstioSidecarStatusAnnotation(pod)
}

func HasIstioSidecarStatusAnnotation(pod v1.Pod) bool {
	_, exists := pod.Annotations["sidecar.istio.io/status"]
	return exists
}

func IsPodReady(pod v1.Pod) bool {
	isMarkedForDeletion := pod.DeletionTimestamp != nil
	return !isMarkedForDeletion && isPodRunning(pod) && hasTrueStatusConditions(pod)
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
	c := pod.Spec.Containers
	c = append(c, pod.Spec.InitContainers...)
	for _, container := range c {
		if isContainerIstioSidecar(container) && !expectedImage.matchesImageIn(container) {
			return true
		}
	}
	return false
}

const (
	istioProxyCPULimitName       = "sidecar.istio.io/proxyCPULimit"
	istioProxyMemoryLimitName    = "sidecar.istio.io/proxyMemoryLimit"
	istioProxyCPURequestsName    = "sidecar.istio.io/proxyCPU"
	istioProxyMemoryRequestsName = "sidecar.istio.io/proxyMemory"
)

func hasDifferentSidecarResources(pod v1.Pod, expectedResources v1.ResourceRequirements) bool {
	// Override expected resources with annotations if they exist
	// In case of parsing error, this function will return false to avoid restart, as
	// istiod injection mutating webhook will reject the pod anyway
	if pod.Annotations != nil {
		if cpuLimit, found := pod.Annotations[istioProxyCPULimitName]; found {
			l, err := resource.ParseQuantity(cpuLimit)
			if err != nil {
				return false
			}
			expectedResources.Limits[v1.ResourceCPU] = l
		}
		if memoryLimit, found := pod.Annotations[istioProxyMemoryLimitName]; found {
			l, err := resource.ParseQuantity(memoryLimit)
			if err != nil {
				return false
			}
			expectedResources.Limits[v1.ResourceMemory] = l
		}
		if cpuRequest, found := pod.Annotations[istioProxyCPURequestsName]; found {
			r, err := resource.ParseQuantity(cpuRequest)
			if err != nil {
				return false
			}
			expectedResources.Requests[v1.ResourceCPU] = r
		}
		if memoryRequest, found := pod.Annotations[istioProxyMemoryRequestsName]; found {
			r, err := resource.ParseQuantity(memoryRequest)
			if err != nil {
				return false
			}
			expectedResources.Requests[v1.ResourceMemory] = r
		}
	}
	for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
		if isContainerIstioSidecar(container) && !containerHasResources(container, expectedResources) {
			return true
		}
	}
	return false
}

func containerHasResources(container v1.Container, expectedResources v1.ResourceRequirements) bool {
	equalCPURequests := container.Resources.Requests.Cpu().Equal(*expectedResources.Requests.Cpu())
	equalMemoryRequests := container.Resources.Requests.Memory().Equal(*expectedResources.Requests.Memory())
	equalCPULimits := container.Resources.Limits.Cpu().Equal(*expectedResources.Limits.Cpu())
	equalMemoryLimits := container.Resources.Limits.Memory().Equal(*expectedResources.Limits.Memory())

	return equalCPURequests && equalMemoryRequests && equalCPULimits && equalMemoryLimits
}

func isContainerIstioSidecar(container v1.Container) bool {
	return istioSidecarName == container.Name
}
