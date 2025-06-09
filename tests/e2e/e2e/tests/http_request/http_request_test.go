package http_request

import (
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/http/no_auth"
	"github.com/stretchr/testify/require"
	"io"
	"testing"
)

func TestHTTPRequest(t *testing.T) {
	t.Parallel()
	testExecutor := executor.DefaultExecutor(t, "http_request")
	defer testExecutor.Cleanup()
	httpRequest := &no_auth.Request{
		URL:    "https://example.com",
		Method: "GET",
		Headers: map[string]string{
			"Accept":     "application/json",
			"User-Agent": "E2E Test Client",
		},
	}
	err := testExecutor.RunStep(httpRequest)
	require.NoError(t, err)
	require.NotEmpty(t, httpRequest.Response.Load(), "Expected a non-empty response from the HTTP request")
	response := httpRequest.Response.Load()
	bdy, err := io.ReadAll(response.Body)
	require.NoError(t, err, "Failed to read response body")
	fmt.Printf("Response Status: %s\n", response.Status)
	fmt.Printf("Response Body: %s\n", bdy)
}
