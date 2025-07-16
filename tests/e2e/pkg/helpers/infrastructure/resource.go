package infrastructure

import (
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

func CreateResource(t *testing.T, resourceTemplate string, resource k8s.Object, opts ...decoder.DecodeOption) (k8s.Object, error) {
	t.Helper()
	r, err := ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return nil, err
	}

	err = decoder.DecodeString(resourceTemplate, resource, opts...)
	if err != nil {
		t.Logf("Failed to decode resource template %s: %v", resourceTemplate, err)
		return nil, err
	}

	t.Logf("Creating %s/%s: name=\"%s\" namespace=\"%s\"",
		resource.GetObjectKind().GroupVersionKind().Kind,
		resource.GetObjectKind().GroupVersionKind().Version,
		resource.GetName(),
		resource.GetNamespace())

	setup.DeclareCleanup(t, func() {
		cleanupResource := resource.DeepCopyObject().(k8s.Object)
		err := decoder.DecodeString(resourceTemplate, cleanupResource, opts...)
		if err != nil {
			t.Logf("Failed to decode cleanup resource template %s: %v", resourceTemplate, err)
			return
		}

		t.Logf("Cleaning up %s/%s: name=\"%s\" namespace=\"%s\"",
			cleanupResource.GetObjectKind().GroupVersionKind().Kind,
			cleanupResource.GetObjectKind().GroupVersionKind().Version,
			cleanupResource.GetName(),
			cleanupResource.GetNamespace())
		err = r.Delete(setup.GetCleanupContext(), resource)
		if err != nil {
			t.Logf("Failed to delete resource %s: %v", resource.GetName(), err)
			return
		}
	})

	return resource, r.Create(t.Context(), resource)
}
