package createpod_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/ns"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
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
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)

	require.NoError(t, ns.CreateNamespace(t, namespaceName, cfg))

	t.Run("test", func(t *testing.T) {
		t.Parallel()
		r, err := resources.New(helpers.WrapTestLog(t, cfg.Client().RESTConfig()))
		require.NoError(t, err)

		// given
		pod := corev1.Pod{}
		require.NoError(t, decoder.DecodeString(nginxPod, &pod, decoder.MutateNamespace(namespaceName)))

		// when
		require.NoError(t, r.Create(t.Context(), &pod))
		// then
		require.NoError(t, wait.For(conditions.New(r).PodRunning(&pod)))
	})
}
