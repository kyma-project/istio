package transform

import (
	"k8s.io/apimachinery/pkg/api/meta"
	toolscache "k8s.io/client-go/tools/cache"
)

// StripForRestart returns a cache.TransformFunc that removes object fields never
// read during predicate evaluation. It clears managedFields and ownerReferences
// while retaining finalizers, name, namespace, labels and annotations.
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

// StripForRestartKeepOwnerRefs preserves OwnerReferences
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
