package httprequest_test

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/helpers/http"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestHTTPRequest(t *testing.T) {
	t.Run("HTTP Request Test", func(t *testing.T) {
		t.Parallel()

		// given
		request := createGetJsonRequest("http://example.com")

		// when
		response, err := http.DoRequest(t, request)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, response, "Expected a non-empty response from the HTTP request")

		require.NoError(t, err, "Failed to read response body")
		t.Logf("Response Status: %s\n", response.Status)
		t.Logf("Response Body: %s\n", response.Body)
	})
}

func createGetJsonRequest(url string) *http.Request {
	return &http.Request{
		URL:    url,
		Method: "GET",
		Headers: map[string]string{
			"Accept":     "application/json",
			"User-Agent": "E2E Test Client",
		},
	}
}
