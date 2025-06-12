package httprequest_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/http/noauth"
)

func TestHTTPRequest(t *testing.T) {
	t.Parallel()
	testExecutor := executor.NewExecutor(t)
	defer testExecutor.Cleanup()

	httpRequest := &noauth.Request{
		URL:    "https://example.com",
		Method: "GET",
		Headers: map[string]string{
			"Accept":     "application/json",
			"User-Agent": "E2E Test Client",
		},
	}
	err := testExecutor.RunStep(httpRequest)
	require.NoError(t, err)
	require.NotEmpty(t, httpRequest.Response, "Expected a non-empty response from the HTTP request")
	response := httpRequest.Response
	require.NoError(t, err, "Failed to read response body")
	t.Logf("Response Status: %s\n", response.Status)
	t.Logf("Response Body: %s\n", string(httpRequest.ResponseBody))
}
