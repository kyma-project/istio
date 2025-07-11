package httprequest_test

import (
	"net/http"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers"
	"github.com/stretchr/testify/require"
)

func TestHTTPRequest(t *testing.T) {
	t.Run("HTTP Request Test", func(t *testing.T) {
		t.Parallel()

		// given
		c := helpers.NewHTTPClient(t)
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
}
