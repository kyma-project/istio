package createpod_test

import (
	_ "embed"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

//go:embed pod.yaml
var nginxPod []byte

func TestPodCreation(t *testing.T) {
	testId := envconf.RandomName("test", 16)
	namespaceName := "ns-" + testId

	require.NoError(t, infrahelpers.CreateNamespace(t, namespaceName))

	t.Run("test", func(t *testing.T) {
		t.Parallel()
		r := infrahelpers.ResourcesClient(t)

		// given
		var pod corev1.Pod

		// when
		resource := infrahelpers.CreateResource(t, string(nginxPod), &pod, decoder.MutateNamespace(namespaceName))

		// then
		require.NoError(t, wait.For(conditions.New(r).PodRunning(resource)))
	})
}
