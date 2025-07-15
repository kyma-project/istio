package createpod_test

import (
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

var nginxPod = `apiVersion: v1
kind: Pod
metadata:
  labels:
    run: nginx
  name: nginx
spec:
  containers:
  - image: nginx:latest
    name: nginx
    ports:
      - containerPort: 80
        name: http
  dnsPolicy: ClusterFirst
  restartPolicy: Always
`

func TestPodCreation(t *testing.T) {
	testId := envconf.RandomName("test", 16)
	namespaceName := "ns-" + testId

	require.NoError(t, infrahelpers.CreateNamespace(t, namespaceName))

	t.Run("test", func(t *testing.T) {
		t.Parallel()
		r := infrahelpers.ResourcesClient(t)

		// given
		pod := corev1.Pod{}
		require.NoError(t, decoder.DecodeString(nginxPod, &pod, decoder.MutateNamespace(namespaceName)))
		setup.DeclareCleanup(t, func() {
			t.Log("Cleaning up pod after the test")
			require.NoError(t, r.Delete(setup.GetCleanupContext(), &pod))
		})

		// when
		require.NoError(t, r.Create(t.Context(), &pod))
		// then
		require.NoError(t, wait.For(conditions.New(r).PodRunning(&pod)))
	})
}
