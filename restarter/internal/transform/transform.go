package transform

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	toolscache "k8s.io/client-go/tools/cache"
)

// StripForRestart returns a cache.TransformFunc that removes object fields never
// read during predicate evaluation. It retains finalizers, name, namespace,
// labels and annotations.
func StripForRestart() toolscache.TransformFunc {
	return func(obj any) (any, error) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			// Not a metav1.Object (e.g. a cache.DeletedFinalStateUnknown tombstone); leave it untouched.
			return obj, nil
		}

		accessor.SetManagedFields(nil)
		accessor.SetOwnerReferences(nil)

		return obj, nil
	}
}

// StripForRestartKeepOwnerRefs returns a cache.TransformFunc that removes object fields never
// read during predicate evaluation. It retains finalizers, name, namespace, labels,
// annotations and owner references.
func StripForRestartKeepOwnerRefs() toolscache.TransformFunc {
	return func(obj any) (any, error) {
		accessor, err := meta.Accessor(obj)
		if err != nil {
			return obj, nil
		}

		accessor.SetManagedFields(nil)

		return obj, nil
	}
}

func minimalContainers(cs []corev1.Container) []corev1.Container {
	out := make([]corev1.Container, len(cs))
	for i, c := range cs {
		out[i] = corev1.Container{
			Name:      c.Name,
			Image:     c.Image,
			Resources: c.Resources,
		}
	}
	return out
}

// StripForPod returns a cache.TransformFunc that trims cached Pod
// objects to only the fields the restarter actually reads.
//
// Fields kept:
//   - metadata.name / metadata.namespace
//   - metadata.ownerReferences
//   - metadata.annotations
//   - spec.initContainers / spec.containers: each container is reduced to Name,
//     Image, and Resources only
func StripForPod() toolscache.TransformFunc {
	return func(obj any) (any, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return obj, nil
		}

		pod.ManagedFields = nil
		pod.Status = corev1.PodStatus{}
		pod.Spec = corev1.PodSpec{
			InitContainers: minimalContainers(pod.Spec.InitContainers),
			Containers:     minimalContainers(pod.Spec.Containers),
		}

		return pod, nil
	}
}
