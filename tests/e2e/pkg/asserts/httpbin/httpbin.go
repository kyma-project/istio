package httpbin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/wait"
)

// HeadersResponse represents the JSON response from httpbin's /headers endpoint
type HeadersResponse struct {
	Headers map[string][]string `json:"headers"`
}

// HeaderAssertion defines an assertion for a specific header
type HeaderAssertion struct {
	Name           string
	ExpectedValues []string
	ShouldExist    bool
}

type AssertOptions struct {
	HeaderAssertions []HeaderAssertion
	Timeout          time.Duration
	Interval         time.Duration
}

type AssertOption func(*AssertOptions)

// WithHeaderValue asserts that a header exists with specific values
func WithHeaderValue(headerName string, expectedValues ...string) AssertOption {
	return func(o *AssertOptions) {
		o.HeaderAssertions = append(o.HeaderAssertions, HeaderAssertion{
			Name:           headerName,
			ExpectedValues: expectedValues,
			ShouldExist:    true,
		})
	}
}

// WithTimeout sets the timeout for the assertion
func WithTimeout(timeout time.Duration) AssertOption {
	return func(o *AssertOptions) {
		o.Timeout = timeout
	}
}

// WithInterval sets the interval for retries
func WithInterval(interval time.Duration) AssertOption {
	return func(o *AssertOptions) {
		o.Interval = interval
	}
}

// AssertHeaders makes a request to httpbin's /headers endpoint and asserts on the reflected headers
func AssertHeaders(t *testing.T, client *http.Client, url string, opts ...AssertOption) {
	t.Helper()

	options := &AssertOptions{
		Timeout:  30 * time.Second,
		Interval: 2 * time.Second,
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

		t.Logf("Making HTTP request to httpbin /headers endpoint: %s", url)

		resp, err := client.Do(req)
		if err != nil {
			t.Logf("Request failed: %v", err)
			return false, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Logf("Expected status code 200, got %d", resp.StatusCode)
			return false, nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Logf("Failed to read response body: %v", err)
			return false, err
		}

		var headersResp HeadersResponse
		if err := json.Unmarshal(body, &headersResp); err != nil {
			t.Logf("Failed to parse JSON response: %v, body: %s", err, string(body))
			return false, err
		}

		t.Logf("Received headers from httpbin: %v", headersResp.Headers)

		// Perform header assertions
		for _, assertion := range options.HeaderAssertions {
			actualValues, exists := headersResp.Headers[assertion.Name]

			if assertion.ShouldExist && !exists {
				t.Logf("Expected header %q to exist, but it was not found", assertion.Name)
				return false, nil
			}

			if !assertion.ShouldExist && exists {
				t.Logf("Expected header %q to not exist, but it was found with values: %v", assertion.Name, actualValues)
				return false, nil
			}

			if assertion.ShouldExist && len(assertion.ExpectedValues) > 0 {
				if len(actualValues) != len(assertion.ExpectedValues) {
					t.Logf("Header %q: expected %d values %v, got %d values %v",
						assertion.Name, len(assertion.ExpectedValues), assertion.ExpectedValues,
						len(actualValues), actualValues)
					return false, nil
				}

				for i, expectedValue := range assertion.ExpectedValues {
					if actualValues[i] != expectedValue {
						t.Logf("Header %q[%d]: expected %q, got %q", assertion.Name, i, expectedValue, actualValues[i])
						return false, nil
					}
				}
			}
		}

		return true, nil
	}, wait.WithTimeout(options.Timeout), wait.WithInterval(options.Interval))

	require.NoError(t, err, "Httpbin headers assertion failed")
}
