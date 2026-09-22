package proxystatsassert

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"

	proxystatshelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/proxy_stats"
)

type AssertOptions struct {
	Timeout  time.Duration
	Interval time.Duration
}

type AssertOption func(*AssertOptions)

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

func AssertOutlierDetectionMetric(t *testing.T, r *resources.Resources, podName, namespace, host string, expectedCount int, opts ...AssertOption) {
	t.Helper()
	options := &AssertOptions{
		Timeout:  60 * time.Second,
		Interval: 2 * time.Second,
	}
	for _, opt := range opts {
		opt(options)
	}

	expectedLine := fmt.Sprintf(
		`envoy_cluster_outlier_detection_ejections_active{cluster_name="outbound|8000||%s"} %d`,
		host, expectedCount,
	)

	err := wait.For(func(ctx context.Context) (bool, error) {
		stats := proxystatshelper.GetProxyStats(t, r, podName, namespace)
		if strings.Contains(stats, expectedLine) {
			return true, nil
		}
		t.Logf("waiting for %q in %s/%s proxy stats", expectedLine, namespace, podName)
		return false, nil
	}, wait.WithTimeout(options.Timeout), wait.WithInterval(options.Interval), wait.WithContext(t.Context()))

	require.NoError(t, err,
		"metric %q not found in %s/%s proxy stats within timeout", expectedLine, namespace, podName)
}

func AssertHTTPStatusCode(t *testing.T, r *resources.Resources, podName, namespace, url, host, expectedStatus string, opts ...AssertOption) {
	t.Helper()
	options := &AssertOptions{
		Timeout:  60 * time.Second,
		Interval: 2 * time.Second,
	}
	for _, opt := range opts {
		opt(options)
	}

	err := wait.For(func(ctx context.Context) (bool, error) {
		statusCode, err := proxystatshelper.GetHTTPStatusCode(t, r, podName, namespace, url, host)
		if err != nil {
			t.Logf("failed to get HTTP status from %s via %s/%s, retrying: %v", url, namespace, podName, err)
			return false, nil
		}
		if statusCode != expectedStatus {
			t.Logf("expected HTTP %s from %s via %s/%s, got %s", expectedStatus, url, namespace, podName, statusCode)
			return false, nil
		}
		return true, nil
	}, wait.WithTimeout(options.Timeout), wait.WithInterval(options.Interval), wait.WithContext(t.Context()))

	require.NoError(t, err,
		"expected HTTP %s from %s via %s/%s within timeout", expectedStatus, url, namespace, podName)
}
