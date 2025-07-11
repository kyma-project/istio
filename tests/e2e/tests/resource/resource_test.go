package resource_test

import (
	"os"

	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func TestCreateResource(t *testing.T) {
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)

	testdata := os.DirFS("testdata")

	r, err := resources.New(helpers.WrapTestLog(t, cfg.Client().RESTConfig()))
	require.NoError(t, err)

	//given
	pod := corev1.Pod{}
	require.NoError(t, decoder.DecodeFile(testdata, "pod.yaml", &pod))

	// when
	require.NoError(t, r.Create(t.Context(), &pod))

	// then
	require.NoError(t, wait.For(conditions.New(r).PodRunning(&pod)))
}
