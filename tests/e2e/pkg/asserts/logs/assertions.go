package logsassert

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/log"
)

type LogAssertOptions struct {
	RequiredEntries []string
	Container       string
	Timeout         time.Duration
	Interval        time.Duration
}

type LogAssertOption func(*LogAssertOptions)

// WithRequiredEntries sets the required log entries that must be present
func WithRequiredEntries(entries ...string) LogAssertOption {
	return func(o *LogAssertOptions) {
		o.RequiredEntries = append(o.RequiredEntries, entries...)
	}
}

// WithContainer sets the container name to check logs from
func WithContainer(container string) LogAssertOption {
	return func(o *LogAssertOptions) {
		o.Container = container
	}
}

// WithTimeout sets the timeout for the assertion
func WithTimeout(timeout time.Duration) LogAssertOption {
	return func(o *LogAssertOptions) {
		o.Timeout = timeout
	}
}

// WithInterval sets the interval for the assertion
func WithInterval(interval time.Duration) LogAssertOption {
	return func(o *LogAssertOptions) {
		o.Interval = interval
	}
}

// AssertIstioProxyLogsContain asserts that istio-proxy logs in pods matching the selector contain all required entries
func AssertIstioProxyLogsContain(t *testing.T, c *resources.Resources, labelSelector string, requiredEntries []string, opts ...LogAssertOption) {
	t.Helper()

	options := &LogAssertOptions{
		RequiredEntries: requiredEntries,
		Container:       "istio-proxy",
		Timeout:         30 * time.Second,
		Interval:        2 * time.Second,
	}

	for _, opt := range opts {
		opt(options)
	}

	podList := v1.PodList{}
	err := c.List(t.Context(), &podList, resources.WithLabelSelector(labelSelector))
	require.NoError(t, err, "Failed to list pods")
	require.NotEmpty(t, podList.Items, "No pods found with selector %s", labelSelector)

	for _, pod := range podList.Items {
		t.Logf("Checking logs from pod %s/%s", pod.Namespace, pod.Name)

		logs, err := log.GetLogsFromPodContainer(t, pod.Name, pod.Namespace, options.Container)
		require.NoError(t, err, "Failed to get logs from pod %s/%s container %s", pod.Namespace, pod.Name, options.Container)

		logsStr := string(logs)
		for _, entry := range options.RequiredEntries {
			require.Containsf(t, logsStr, entry, "Log entry %q not found in logs from pod %s/%s", entry, pod.Namespace, pod.Name)
		}
	}
}

// AssertContainerLogContainsWithRetry asserts that container logs contain a specific string with retry logic
func AssertContainerLogContainsWithRetry(t *testing.T, c *resources.Resources, labelSelector, namespace, containerName, expectedLog string, opts ...LogAssertOption) {
	t.Helper()

	options := &LogAssertOptions{
		Container: containerName,
		Timeout:   30 * time.Second,
		Interval:  2 * time.Second,
	}

	for _, opt := range opts {
		opt(options)
	}

	podList := v1.PodList{}
	err := c.List(t.Context(), &podList, resources.WithLabelSelector(labelSelector), resources.WithFieldSelector("metadata.namespace="+namespace))
	require.NoError(t, err, "Failed to list pods")
	require.NotEmpty(t, podList.Items, "No pods found with selector %s in namespace %s", labelSelector, namespace)

	for _, pod := range podList.Items {
		t.Logf("Checking logs from pod %s/%s container %s", pod.Namespace, pod.Name, containerName)

		err := wait.For(func(ctx context.Context) (done bool, err error) {
			logs, err := log.GetLogsFromPodContainer(t, pod.Name, pod.Namespace, containerName)
			if err != nil {
				t.Logf("Failed to get logs from pod %s/%s, retrying: %v", pod.Namespace, pod.Name, err)
				return false, nil
			}

			if len(logs) == 0 {
				t.Logf("No logs found in pod %s/%s yet, retrying...", pod.Namespace, pod.Name)
				return false, nil
			}

			logsStr := string(logs)
			if !strings.Contains(logsStr, expectedLog) {
				t.Logf("Expected log %q not found in pod %s/%s yet, retrying...", expectedLog, pod.Namespace, pod.Name)
				return false, nil
			}

			return true, nil
		}, wait.WithTimeout(options.Timeout), wait.WithInterval(options.Interval))

		require.NoError(t, err, "Expected log %q not found in pod %s/%s container %s within timeout", expectedLog, pod.Namespace, pod.Name, containerName)
	}
}
