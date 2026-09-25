package proxystatsassert

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
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

// AssertOutlierDetectionMetric waits until the ejections_active gauge for the
// given host cluster on the specified pod reaches expectedCount.
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
		stats, getErr := proxystatshelper.GetProxyStats(t, r, podName, namespace)
		if getErr != nil {
			t.Logf("failed to get proxy stats from %s/%s, retrying: %v", namespace, podName, getErr)
			return false, nil
		}
		if strings.Contains(stats, expectedLine) {
			return true, nil
		}
		t.Logf("waiting for %q in %s/%s proxy stats", expectedLine, namespace, podName)
		return false, nil
	}, wait.WithTimeout(options.Timeout), wait.WithInterval(options.Interval), wait.WithContext(t.Context()))

	require.NoError(t, err,
		"metric %q not found in %s/%s proxy stats within timeout", expectedLine, namespace, podName)
}

// AssertHTTPStatusCode waits until a curl from the given pod to url returns expectedStatus.
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

// AssertProxyStatsAbsent asserts that the given substring is not present in the proxy stats of the specified pod.
func AssertProxyStatsAbsent(t *testing.T, r *resources.Resources, podName, namespace, substring string) {
	t.Helper()

	stats, err := proxystatshelper.GetProxyStats(t, r, podName, namespace)
	require.NoError(t, err, "failed to get proxy stats from %s/%s", namespace, podName)
	require.NotContains(t, stats, substring,
		"expected %q to be absent in %s/%s proxy stats", substring, namespace, podName)
}

// AssertProxyStatsPresent waits until the given substring is present in the
// proxy stats of the specified pod, retrying until timeout.
func AssertProxyStatsPresent(t *testing.T, r *resources.Resources, podName, namespace, substring string, opts ...AssertOption) {
	t.Helper()
	options := &AssertOptions{
		Timeout:  60 * time.Second,
		Interval: 2 * time.Second,
	}
	for _, opt := range opts {
		opt(options)
	}

	err := wait.For(func(ctx context.Context) (bool, error) {
		stats, getErr := proxystatshelper.GetProxyStats(t, r, podName, namespace)
		if getErr != nil {
			t.Logf("failed to get proxy stats from %s/%s, retrying: %v", namespace, podName, getErr)
			return false, nil
		}
		if !strings.Contains(stats, substring) {
			t.Logf("waiting for %q to appear in %s/%s proxy stats", substring, namespace, podName)
			return false, nil
		}
		return true, nil
	}, wait.WithTimeout(options.Timeout), wait.WithInterval(options.Interval), wait.WithContext(t.Context()))

	require.NoError(t, err,
		"metric %q not found in %s/%s proxy stats within timeout", substring, namespace, podName)
}

// AssertAtLeastOneIngressGatewayPodHasEjection asserts that at least one running
// ingress gateway pod reports ejections_active >= 1 for the given host cluster.
func AssertAtLeastOneIngressGatewayPodHasEjection(t *testing.T, r *resources.Resources, host string) {
	t.Helper()

	podList := &corev1.PodList{}
	err := r.List(t.Context(), podList,
		resources.WithLabelSelector("app=istio-ingressgateway"),
		resources.WithFieldSelector("metadata.namespace=istio-system"),
	)
	require.NoError(t, err, "failed to list ingress gateway pods")

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		stats, err := proxystatshelper.GetProxyStats(t, r, pod.Name, "istio-system")
		require.NoError(t, err, "failed to get proxy stats from istio-system/%s", pod.Name)

		if ejections, _ := countOutlierEjectionsActive(stats, host); ejections >= 1 {
			return
		}
	}

	t.Fatal("expected at least one ingress gateway pod to report ejections_active>=1")
}

// AssertAllIngressGatewayPodsHaveEjectionMetric asserts that every running
// ingress gateway pod exposes the outlier detection ejections_active metric for
// the given host cluster.
func AssertAllIngressGatewayPodsHaveEjectionMetric(t *testing.T, r *resources.Resources, host string) {
	t.Helper()

	podList := &corev1.PodList{}
	err := r.List(t.Context(), podList,
		resources.WithLabelSelector("app=istio-ingressgateway"),
		resources.WithFieldSelector("metadata.namespace=istio-system"),
	)
	require.NoError(t, err, "failed to list ingress gateway pods")

	checked := 0
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		checked++
		stats, err := proxystatshelper.GetProxyStats(t, r, pod.Name, "istio-system")
		require.NoError(t, err, "failed to get proxy stats from istio-system/%s", pod.Name)

		_, found := countOutlierEjectionsActive(stats, host)
		require.True(t, found,
			"ingress gateway pod %s does not expose ejections_active metric for host %s", pod.Name, host)
	}

	require.Greater(t, checked, 0, "no running ingress gateway pods found")
}

// AssertEjectionStateMatchesUpstream5xxMetric asserts that for each running ingress
// gateway pod, its active ejection state matches whether it observed at least
// threshold upstream 5xx responses for the given host cluster.
func AssertEjectionStateMatchesUpstream5xxMetric(t *testing.T, r *resources.Resources, host string, threshold int) {
	t.Helper()

	podList := &corev1.PodList{}
	err := r.List(t.Context(), podList,
		resources.WithLabelSelector("app=istio-ingressgateway"),
		resources.WithFieldSelector("metadata.namespace=istio-system"),
	)
	require.NoError(t, err, "failed to list ingress gateway pods")

	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			continue
		}
		stats, err := proxystatshelper.GetProxyStats(t, r, pod.Name, "istio-system")
		require.NoError(t, err, "failed to get proxy stats from istio-system/%s", pod.Name)

		ejections, _ := countOutlierEjectionsActive(stats, host)
		upstream5xx := countUpstream5xxResponses(stats, host)

		hasEnough5xx := upstream5xx >= threshold
		hasActiveEjection := ejections >= 1

		require.Equal(t, hasEnough5xx, hasActiveEjection,
			"pod %s: activeEjection=%t, upstream5xx=%d (threshold %d)",
			pod.Name, hasActiveEjection, upstream5xx, threshold)
	}
}

// countOutlierEjectionsActive returns the ejections_active gauge value for the
// given host cluster and whether the metric line was found at all.
func countOutlierEjectionsActive(stats, host string) (count int, found bool) {
	for line := range strings.SplitSeq(stats, "\n") {
		if !strings.Contains(line, "envoy_cluster_outlier_detection_ejections_active") ||
			!strings.Contains(line, host) {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) == 0 {
			continue
		}
		_, err := fmt.Sscanf(parts[len(parts)-1], "%d", &count)
		if err == nil {
			return count, true
		}
	}
	return 0, false
}

// countUpstream5xxResponses sums all envoy_cluster_upstream_rq lines for the
// given host cluster that carry response_code_class="5xx".
func countUpstream5xxResponses(stats, host string) int {
	total := 0
	for line := range strings.SplitSeq(stats, "\n") {
		if metricName(line) != "envoy_cluster_upstream_rq" ||
			!strings.Contains(line, host) ||
			!strings.Contains(line, `response_code_class="5xx"`) {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 0 {
			count := 0
			_, _ = fmt.Sscanf(parts[len(parts)-1], "%d", &count)
			total += count
		}
	}
	return total
}

// metricName returns the metric name portion of a Prometheus text line,
// stripping any label set and surrounding whitespace.
func metricName(line string) string {
	name, _, _ := strings.Cut(line, "{")
	return strings.TrimSpace(name)
}
