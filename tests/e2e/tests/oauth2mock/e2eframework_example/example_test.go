package e2eframework_example

import (
	"context"
	"os"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/assess"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup/oauth2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/features"
)

func TestExample(t *testing.T) {
	testdata := os.DirFS("testdata")

	feat := features.New("example").
		Setup(assess.SetupIstioStep(testdata)).
		Assess("oauth2-mock is running", func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
			m, ok := oauth2.FromContext(ctx)
			require.True(t, ok)
			assert.Equal(t, "http://mock-oauth2-server.oauth2-mock.svc.cluster.local", m.IssuerURL)
			assert.Equal(t, "oauth2-mock.local.kyma.dev", m.Subdomain)
			assert.Equal(t, "https://oauth2-mock.local.kyma.dev/oauth2/token", m.TokenURL)
			return ctx
		}).
		Assess("connection to https://kyma-project.io is working", assess.RunCurlClusterStep("curl -sSL https://kyma-project.io")).
		Assess("connection to https://httpbin.org/headers is working", assess.RunCurlClusterStep("curl -sSL https://httpbin.org/headers")).
		Teardown(assess.TeardownIstioStep(testdata)).
		Feature()
	testEnv.Test(t, feat)
}
