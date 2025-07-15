package infrastructure

import (
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"github.com/stretchr/testify/require"
)

func CreateResource(t *testing.T, resourceTemplate string, resource k8s.Object, opts ...decoder.DecodeOption) k8s.Object {
	t.Helper()
	r := ResourcesClient(t)
	require.NoError(t, decoder.DecodeString(resourceTemplate, resource, opts...))

	t.Logf("Creating %s/%s: name=\"%s\" namespace=\"%s\"",
		resource.GetObjectKind().GroupVersionKind().Kind,
		resource.GetObjectKind().GroupVersionKind().Version,
		resource.GetName(),
		resource.GetNamespace())

	setup.DeclareCleanup(t, func() {
		cleanupResource := resource.DeepCopyObject().(k8s.Object)
		require.NoError(t, decoder.DecodeString(resourceTemplate, cleanupResource, opts...))
		t.Logf("Cleaning up %s/%s: name=\"%s\" namespace=\"%s\"",
			cleanupResource.GetObjectKind().GroupVersionKind().Kind,
			cleanupResource.GetObjectKind().GroupVersionKind().Version,
			cleanupResource.GetName(),
			cleanupResource.GetNamespace())
		require.NoError(t, r.Delete(setup.GetCleanupContext(), resource))
	})

	require.NoError(t, r.Create(t.Context(), resource))
	return resource
}
