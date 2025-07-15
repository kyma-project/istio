package resource_test

import (
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"os"

	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

func TestCreateResource(t *testing.T) {
	r := infrahelpers.ResourcesClient(t)

	//given
	pod := corev1.Pod{}
	testdata := os.DirFS("testdata")
	require.NoError(t, decoder.DecodeFile(testdata, "pod.yaml", &pod))
	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up pod after the test")
		require.NoError(t, r.Delete(setup.GetCleanupContext(), &pod))
	})

	// when
	require.NoError(t, r.Create(t.Context(), &pod))

	// then
	require.NoError(t, wait.For(conditions.New(r).PodRunning(&pod)))
}
