package v2

import (
	"strings"

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
	expectedTag string
}

func (i *SidecarImageChanged) Name() string { return "SidecarImageChanged" }

// Evaluate returns Restart when the sidecar image tag no longer matches the
// expected one, and Continue otherwise. When expectedTag is empty the expected
// version is unknown and the rule never forces a restart.
func (i *SidecarImageChanged) Evaluate(obj Object) Decision {
	if i.expectedTag == "" {
		return Continue
	}
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		// this is not a pod, bail out, don't stop
		return Continue
	}
	if pod.Annotations[istioSidecarCustomImageAnnotation] != "" {
		return Continue
	}
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		if c.Name == istioSidecarName && sidecarImageTag(c.Image) != i.expectedTag {
			return Restart
		}
	}
	// nothing to restart, continue
	return Continue
}

// sidecarImageTag extracts the tag from a container image reference.
// It handles:
//   - digest-only refs (foo@sha256:abc) -> ""
//   - refs with both tag and digest (foo:1.2.3@sha256:abc) -> "1.2.3"
//   - refs with a registry port (localhost:5000/ns/img:tag) -> "tag"
//   - plain refs (registry/name:tag) -> "tag"
func sidecarImageTag(ref string) string {
	// Strip digest suffix first so a `:` inside the digest hash is not confused
	// with the tag separator.
	if idx := strings.Index(ref, "@"); idx != -1 {
		ref = ref[:idx]
	}
	// Only consider the last path segment when searching for the tag colon, so
	// a port in the registry host (e.g. localhost:5000) is not mistaken for the
	// tag separator.
	lastSlash := strings.LastIndex(ref, "/")
	lastSegment := ref[lastSlash+1:]
	colonIdx := strings.LastIndex(lastSegment, ":")
	if colonIdx == -1 {
		// No tag present (digest-only after stripping, or bare name).
		return ""
	}
	return lastSegment[colonIdx+1:]
}

// NewSidecarImageChangedRule returns a Evaluator that restarts workloads whose
// sidecar image tag differs from expectedTag. When expectedTag is empty the
// evaluator never forces a restart.
func NewSidecarImageChangedRule(expectedTag string) Evaluator {
	return &SidecarImageChanged{expectedTag: expectedTag}
}

// SidecarResourcesChanged is a Evaluator that detects Istio proxy sidecar resource drift.
type SidecarResourcesChanged struct {
	req corev1.ResourceRequirements
}

func (i *SidecarResourcesChanged) Name() string { return "SidecarResourcesChanged" }

// Evaluate returns Restart when the sidecar's resource requests or limits differ
// from the expected ones, and Continue otherwise.
func (i *SidecarResourcesChanged) Evaluate(obj Object) Decision {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return Continue
	}
	effective, err := ParseResourceAnnotations(pod, i.req)
	if err != nil {
		return Continue
	}
	for _, c := range append(pod.Spec.InitContainers, pod.Spec.Containers...) {
		if c.Name != istioSidecarName {
			continue
		}
		equalCPURequests := c.Resources.Requests.Cpu().Equal(*effective.Requests.Cpu())
		equalMemoryRequests := c.Resources.Requests.Memory().Equal(*effective.Requests.Memory())
		equalCPULimits := c.Resources.Limits.Cpu().Equal(*effective.Limits.Cpu())
		equalMemoryLimits := c.Resources.Limits.Memory().Equal(*effective.Limits.Memory())

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
