package httprequest_test

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/http/noauth"
)

func TestHTTPRequest(t *testing.T) {
	t.Run("HTTP Request Test", func(t *testing.T) {
		t.Parallel()
		testExecutor := executor.NewExecutorWithOptionsFromEnv(t)
		defer testExecutor.Cleanup()

		// given
		httpRequest := &noauth.Request{
			URL:    "https://example.com",
			Method: "GET",
			Headers: map[string]string{
				"Accept":     "application/json",
				"User-Agent": "E2E Test Client",
			},
		}

		// when
		err := testExecutor.RunStep(httpRequest)

		// then
		require.NoError(t, err)
		require.NotEmpty(t, httpRequest.Response, "Expected a non-empty response from the HTTP request")

		response := httpRequest.Response
		require.NoError(t, err, "Failed to read response body")
		logging.Infof(t, "Response Status: %s\n", response.Status)
		logging.Infof(t, "Response Body: %s\n", string(httpRequest.ResponseBody))
	})
}
