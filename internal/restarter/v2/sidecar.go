package v2

import (
	"github.com/kyma-project/istio/operator/internal/images"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	istioSidecarName                  = "istio-proxy"
	istioSidecarCustomImageAnnotation = "sidecar.istio.io/proxyImage"
	istioProxyCPULimitName            = "sidecar.istio.io/proxyCPULimit"
	istioProxyMemoryLimitName         = "sidecar.istio.io/proxyMemoryLimit"
	istioProxyCPURequestsName         = "sidecar.istio.io/proxyCPU"
	istioProxyMemoryRequestsName      = "sidecar.istio.io/proxyMemory"
)

// SidecarImageChanged is a Evaluator that detects Istio proxy sidecar image drift.
type SidecarImageChanged struct {
	expectedImage images.Image
}

// Evaluate returns Restart when the sidecar image no longer matches the expected
// one, and Continue otherwise.
func (i *SidecarImageChanged) Evaluate(obj Object) Decision {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		// this is not a pod, bail out, don't stop
		return Continue
	}
	if pod.Annotations[istioSidecarCustomImageAnnotation] != "" {
		return Continue
	}
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		if c.Name == istioSidecarName && !i.expectedImage.MatchesImageInContainer(c) {
			return Restart
		}
	}
	// nothing to restart, continue
	return Continue
}

// NewSidecarImageChangedRule returns a Evaluator that restarts workloads whose sidecar
// image differs from the expected one.
func NewSidecarImageChangedRule(image images.Image) Evaluator {
	return &SidecarImageChanged{expectedImage: image}
}

// SidecarResourcesChanged is a Evaluator that detects Istio proxy sidecar resource drift.
type SidecarResourcesChanged struct {
	req corev1.ResourceRequirements
}

// Evaluate returns Restart when the sidecar's resource requests or limits differ
// from the expected ones, and Continue otherwise.
func (i *SidecarResourcesChanged) Evaluate(obj Object) Decision {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return Continue
	}
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		if c.Name != istioSidecarName {
			continue
		}
		equalCPURequests := c.Resources.Requests.Cpu().Equal(*i.req.Requests.Cpu())
		equalMemoryRequests := c.Resources.Requests.Memory().Equal(*i.req.Requests.Memory())
		equalCPULimits := c.Resources.Limits.Cpu().Equal(*i.req.Limits.Cpu())
		equalMemoryLimits := c.Resources.Limits.Memory().Equal(*i.req.Limits.Memory())

		if equalCPURequests && equalMemoryRequests && equalCPULimits && equalMemoryLimits {
			return Continue
		}
		return Restart
	}
	return Continue
}

// NewSidecarResourcesChangedRule returns a Evaluator that restarts workloads whose
// sidecar resources differ from the given requirements.
func NewSidecarResourcesChangedRule(reqs corev1.ResourceRequirements) Evaluator {
	return &SidecarResourcesChanged{req: reqs}
}

// ParseResourceAnnotations returns reqs overridden by the workload's Istio sidecar
// resource annotations, or an error if an annotated value is not a valid quantity.
func ParseResourceAnnotations(obj Object, reqs corev1.ResourceRequirements) (corev1.ResourceRequirements, error) {
	var annotations map[string]string
	switch w := obj.(type) {
	case *corev1.Pod:
		annotations = w.Annotations
	case *appsv1.DaemonSet:
		annotations = w.Spec.Template.Annotations
	case *appsv1.Deployment:
		annotations = w.Spec.Template.Annotations
	case *appsv1.StatefulSet:
		annotations = w.Spec.Template.Annotations
	case *appsv1.ReplicaSet:
		annotations = w.Spec.Template.Annotations
	}

	if annotations[istioProxyCPULimitName] == "" && annotations[istioProxyMemoryLimitName] == "" &&
		annotations[istioProxyCPURequestsName] == "" && annotations[istioProxyMemoryRequestsName] == "" {
		return reqs, nil
	}

	// Reset requirements before applying annotated values, matching how Istio
	// defaults them.
	reqs.Limits = corev1.ResourceList{}
	reqs.Requests = corev1.ResourceList{}

	if cpuLimit, found := annotations[istioProxyCPULimitName]; found {
		l, err := resource.ParseQuantity(cpuLimit)
		if err != nil {
			return corev1.ResourceRequirements{}, err
		}
		reqs.Limits[corev1.ResourceCPU] = l
		if annotations[istioProxyCPURequestsName] == "" {
			reqs.Requests[corev1.ResourceCPU] = l
		}
	}
	if memoryLimit, found := annotations[istioProxyMemoryLimitName]; found {
		l, err := resource.ParseQuantity(memoryLimit)
		if err != nil {
			return corev1.ResourceRequirements{}, err
		}
		reqs.Limits[corev1.ResourceMemory] = l
		if annotations[istioProxyMemoryRequestsName] == "" {
			reqs.Requests[corev1.ResourceMemory] = l
		}
	}
	if cpuRequest, found := annotations[istioProxyCPURequestsName]; found {
		r, err := resource.ParseQuantity(cpuRequest)
		if err != nil {
			return corev1.ResourceRequirements{}, err
		}
		reqs.Requests[corev1.ResourceCPU] = r
	}
	if memoryRequest, found := annotations[istioProxyMemoryRequestsName]; found {
		r, err := resource.ParseQuantity(memoryRequest)
		if err != nil {
			return corev1.ResourceRequirements{}, err
		}
		reqs.Requests[corev1.ResourceMemory] = r
	}
	return reqs, nil
}
