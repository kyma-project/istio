package oauth2mock

import (
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/oauth2mock"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/testid"
	"testing"

	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOauth2FS(t *testing.T) {

	require.NoError(t, modulehelpers.CreateIstioCR(t))

	t.Run("deploying oauth2mock in the default oauth2-mock namespace", func(t *testing.T) {
		t.Parallel()
		m, err := oauth2mock.DeployMock(t, "local.kyma.dev")
		require.NoError(t, err)

		assert.Equal(t, "mock-oauth2-server.oauth2-mock.svc.cluster.local", m.IssuerURL)
		assert.Equal(t, "oauth2-mock.local.kyma.dev", m.Subdomain)
		assert.Equal(t, "https://oauth2-mock.local.kyma.dev/oauth2/token", m.TokenURL)
	})

	t.Run("deploying oauth2mock in a custom namespace", func(t *testing.T) {
		t.Parallel()
		_, testNamespace, err := testid.CreateNamespaceWithRandomID(t, testid.Options{
			Prefix: "custom-oauth2-mock",
		})
		require.NoError(t, err, "Failed to create a test namespace")

		m, err := oauth2mock.DeployMock(t, "local.kyma.dev",
			oauth2mock.Options{Namespace: testNamespace})
		require.NoError(t, err)

		assert.Equal(t, fmt.Sprintf("mock-oauth2-server.%s.svc.cluster.local", testNamespace), m.IssuerURL)
		assert.Equal(t, fmt.Sprintf("%s.local.kyma.dev", testNamespace), m.Subdomain)
		assert.Equal(t, fmt.Sprintf("https://%s.local.kyma.dev/oauth2/token", testNamespace), m.TokenURL)
	})
}
