package oauth2mock

import (
	"net/http"
	"os"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/ns"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup/oauth2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func TestOauth2FS(t *testing.T) {
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	fsys := os.DirFS("testdata")

	resp, err := http.Get("https://github.com/kyma-project/istio/releases/download/1.20.1/istio-manager.yaml")
	require.NoError(t, err)
	defer resp.Body.Close()
	require.NoError(t, ns.CreateNamespace(t, "kyma-system", cfg))
	r, err := resources.New(helpers.WrapTestLog(t, cfg.Client().RESTConfig()))
	require.NoError(t, err)
	require.NoError(t, decoder.DecodeEach(t.Context(), resp.Body, decoder.CreateHandler(r), decoder.MutateNamespace("kyma-system")))
	t.Log("Setting up Istio for the tests")
	require.NoError(t, helpers.SetupIstio(t, fsys, cfg))

	m := oauth2.New("local.kyma.dev")
	require.NoError(t, oauth2.StartMock(t, m, cfg))
	assert.Equal(t, "http://mock-oauth2-server.oauth2-mock.svc.cluster.local", m.IssuerURL)
	assert.Equal(t, "oauth2-mock.local.kyma.dev", m.Subdomain)
	assert.Equal(t, "https://oauth2-mock.local.kyma.dev/oauth2/token", m.TokenURL)
}
