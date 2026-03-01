package httpassert

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/wait"
)

type AssertOptions struct {
	ExpectedStatusCode   int
	ExpectedBodyContains []string
	Timeout              time.Duration
	Interval             time.Duration
}

type AssertOption func(*AssertOptions)

// WithExpectedStatusCode sets the expected HTTP status code
func WithExpectedStatusCode(code int) AssertOption {
	return func(o *AssertOptions) {
		o.ExpectedStatusCode = code
	}
}

func WithExpectedBodyContains(contains ...string) AssertOption {
	return func(o *AssertOptions) {
		o.ExpectedBodyContains = append(o.ExpectedBodyContains, contains...)
	}
}

func WithTimeout(timeout time.Duration) AssertOption {
	return func(o *AssertOptions) {
		o.Timeout = timeout
	}
}

func WithInterval(interval time.Duration) AssertOption {
	return func(o *AssertOptions) {
		o.Interval = interval
	}
}

func AssertResponse(t *testing.T, client *http.Client, url string, opts ...AssertOption) {
	t.Helper()

	options := &AssertOptions{
		ExpectedStatusCode: http.StatusOK,
		Timeout:            30 * time.Second,
		Interval:           2 * time.Second,
	}

	for _, opt := range opts {
		opt(options)
	}

	err := wait.For(func(ctx context.Context) (bool, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			t.Logf("Failed to create request: %v", err)
			return false, err
		}

		t.Logf("Making HTTP request to: %s", url)
		t.Logf("  Host header: %s", req.Host)
		if len(req.Header) > 0 {
			t.Logf("  Custom headers: %v", req.Header)
		}

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Request failed: %v", err)
			return false, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != options.ExpectedStatusCode {
			t.Logf("Expected status code %d, got %d", options.ExpectedStatusCode, resp.StatusCode)
			return false, nil
		}

		if len(options.ExpectedBodyContains) > 0 {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logf("Failed to read response body: %v", err)
				return false, err
			}
			bodyStr := string(body)

			for _, expected := range options.ExpectedBodyContains {
				if !strings.Contains(bodyStr, expected) {
					t.Logf("Expected body to contain %q, got: %s", expected, bodyStr)
					return false, nil
				}
			}
		}

		return true, nil
	}, wait.WithTimeout(options.Timeout), wait.WithInterval(options.Interval))

	require.NoError(t, err, "HTTP assertion failed")
}

func AssertOKResponse(t *testing.T, client *http.Client, url string, opts ...AssertOption) {
	t.Helper()
	opts = append([]AssertOption{WithExpectedStatusCode(http.StatusOK)}, opts...)
	AssertResponse(t, client, url, opts...)
}

func AssertForbiddenResponse(t *testing.T, client *http.Client, url string, opts ...AssertOption) {
	t.Helper()
	opts = append([]AssertOption{WithExpectedStatusCode(http.StatusForbidden)}, opts...)
	AssertResponse(t, client, url, opts...)
}

// AssertConnectionError asserts that a request fails with a connection error (e.g., timeout, connection refused).
// This is useful for testing NetworkPolicy blocking at network level.
func AssertConnectionError(t *testing.T, client *http.Client, url string) {
	t.Helper()

	req, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	_, err = client.Do(req)
	require.Error(t, err, "Request should fail due to connection error (e.g., NetworkPolicy blocking traffic)")
}

