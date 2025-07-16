package httprequest_test

import (
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpincluster"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHTTPRequest(t *testing.T) {
	t.Run("HTTP Request Test", func(t *testing.T) {
		t.Parallel()

		// given
		c := httphelper.NewHTTPClient(t)
		req, err := http.NewRequest("GET", "http://example.com", nil)
		req.Header.Add("User-Agent", "E2E Test Client")
		req.Header.Add("Accept", "application/json")

		// when
		response, err := c.Do(req)
		// then
		require.NoError(t, err)
		require.NotEmpty(t, response, "Expected a non-empty response from the HTTP request")

		require.NoError(t, err, "Failed to read response body")
		t.Logf("Response Status: %s\n", response.Status)
		t.Logf("Response Body: %s\n", response.Body)
	})

	t.Run("Run HTTP request from in-cluster pod", func(t *testing.T) {
		t.Parallel()

		// when
		stdout, stderr, err := httpincluster.RunRequestFromInsideCluster(t, "default", "http://example.com")

		// then
		require.NoError(t, err, "Failed to run HTTP request from in-cluster pod")
		require.NotEmpty(t, stdout, "Expected non-empty stdout from the HTTP request")
		require.Empty(t, stderr, "Expected empty stderr from the HTTP request")
	})
}
