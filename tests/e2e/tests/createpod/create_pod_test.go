package createpod_test

import (
	_ "embed"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/testid"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

//go:embed pod.yaml
var nginxPod []byte

func TestPodCreation(t *testing.T) {
	_, ns, err := testid.CreateNamespaceWithRandomID(t, testid.WithPrefix("create-pod"))
	require.NoError(t, err, "Failed to create test namespace")

	t.Run("test", func(t *testing.T) {
		t.Parallel()
		r, err := infrahelpers.ResourcesClient(t)
		require.NoError(t, err, "Failed to get resources client")

		// given
		var pod corev1.Pod

		// when
		resource, err := infrahelpers.CreateResource(t, string(nginxPod), &pod, decoder.MutateNamespace(ns))
		require.NoError(t, err, "Failed to create pod resource")

		// then
		require.NoError(t, wait.For(conditions.New(r).PodRunning(resource)))
	})
}
