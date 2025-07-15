package oauth2mock

import (
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/oauth2mock"
	"testing"

	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOauth2FS(t *testing.T) {

	require.NoError(t, modulehelpers.CreateIstioCR(t))

	m, err := oauth2mock.DeployMock(t, "local.kyma.dev")
	require.NoError(t, err)

	assert.Equal(t, "mock-oauth2-server.oauth2-mock.svc.cluster.local", m.IssuerURL)
	assert.Equal(t, "oauth2-mock.local.kyma.dev", m.Subdomain)
	assert.Equal(t, "https://oauth2-mock.local.kyma.dev/oauth2/token", m.TokenURL)
}
