package proxy_stats_matcher

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apinetworkingv1 "istio.io/api/networking/v1"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

const (
	targetNamespace = "target"
	sourceNamespace = "source"

	outlierDetectionRegexp = ".*outlier_detection.*"
	httpbinHost            = "httpbin." + targetNamespace + ".svc.cluster.local"
	httpbinURL             = "http://" + httpbinHost + ":8000"

	podReadyTimeout    = 2 * time.Minute
	metricsWaitTimeout = 3 * time.Minute
)

func TestProxyStatsMatcher(t *testing.T) {
	t.Run("Outlier detection metrics are exposed on source proxy when proxyStatsMatcher is configured via pod annotation", func(t *testing.T) {
		// given
		r, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureEvaluationClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, targetNamespace)
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, sourceNamespace, infrahelpers.WithSidecarInjectionEnabled())
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().
			WithNamespace(targetNamespace).
			DeployWithCleanup(t)
		require.NoError(t, err)

		err = createOutlierDetectionDestinationRule(t, r, targetNamespace, httpbinHost)
		require.NoError(t, err)

		// Deploy two curl pods with outlier_detection metrics enabled via annotation.
		// Each pod tracks ejection state independently.
		curl1Name := "curl-1"
		curl2Name := "curl-2"

		err = deployCurlPodWithStatsAnnotation(t, r, curl1Name, sourceNamespace)
		require.NoError(t, err)

		err = deployCurlPodWithStatsAnnotation(t, r, curl2Name, sourceNamespace)
		require.NoError(t, err)

		// when: send 5 consecutive 5xx from curl-1 to cross the ejection threshold
		for _, status := range []int{501, 502, 503, 504, 505} {
			url := fmt.Sprintf("%s/status/%d", httpbinURL, status)
			execCurl(t, r, curl1Name, sourceNamespace, url)
		}

		// then: curl-1 sidecar should report ejections_active == 1
		assertOutlierDetectionMetric(t, r, curl1Name, sourceNamespace, 1)

		// and: curl-2 sidecar should report ejections_active == 0 (independent state)
		assertOutlierDetectionMetric(t, r, curl2Name, sourceNamespace, 0)
	})

	t.Run("Outlier detection metrics are exposed on all proxies when proxyStatsMatcher is configured globally via Istio CR", func(t *testing.T) {
		// given
		r, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureEvaluationClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			WithProxyStatsMatcher([]string{outlierDetectionRegexp}).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, targetNamespace, infrahelpers.IgnoreAlreadyExists())
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, sourceNamespace,
			infrahelpers.WithSidecarInjectionEnabled(),
			infrahelpers.IgnoreAlreadyExists())
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().
			WithNamespace(targetNamespace).
			DeployWithCleanup(t)
		require.NoError(t, err)

		err = createOutlierDetectionDestinationRule(t, r, targetNamespace, httpbinHost)
		require.NoError(t, err)

		// Deploy a curl pod without any annotation — metrics come from the global Istio CR config.
		curlName := "curl-global"
		err = deployCurlPod(t, r, curlName, sourceNamespace, nil)
		require.NoError(t, err)

		// when: trigger ejection
		for _, status := range []int{501, 502, 503, 504, 505} {
			url := fmt.Sprintf("%s/status/%d", httpbinURL, status)
			execCurl(t, r, curlName, sourceNamespace, url)
		}

		// then: outlier_detection stats must be visible via the global proxyStatsMatcher config
		assertOutlierDetectionMetric(t, r, curlName, sourceNamespace, 1)
	})

	t.Run("Outlier detection metrics are not exposed when proxyStatsMatcher is not configured", func(t *testing.T) {
		// given
		r, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureEvaluationClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, targetNamespace, infrahelpers.IgnoreAlreadyExists())
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, sourceNamespace,
			infrahelpers.WithSidecarInjectionEnabled(),
			infrahelpers.IgnoreAlreadyExists())
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().
			WithNamespace(targetNamespace).
			DeployWithCleanup(t)
		require.NoError(t, err)

		err = createOutlierDetectionDestinationRule(t, r, targetNamespace, httpbinHost)
		require.NoError(t, err)

		curlName := "curl-no-stats"
		err = deployCurlPod(t, r, curlName, sourceNamespace, nil)
		require.NoError(t, err)

		// when: trigger ejections
		for _, status := range []int{501, 502, 503, 504, 505} {
			url := fmt.Sprintf("%s/status/%d", httpbinURL, status)
			execCurl(t, r, curlName, sourceNamespace, url)
		}

		// then: outlier_detection metrics must NOT appear in proxy stats
		stats := getProxyStats(t, r, curlName, sourceNamespace)
		assert.NotContains(t, stats, "outlier_detection",
			"expected no outlier_detection metrics without proxyStatsMatcher")
	})

	t.Run("Per-workload annotation replaces global proxyStatsMatcher instead of extending it", func(t *testing.T) {
		// The docs state that proxy.istio.io/config replaces the global proxyStatsMatcher for
		// that workload. This test verifies the replacement semantics: a pod annotated with
		// pattern A does not inherit pattern B from the global Istio CR.

		// given
		r, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureEvaluationClusterProfile(t)
		require.NoError(t, err)

		// Global config enables outlier_detection stats.
		_, err = modulehelpers.NewIstioCRBuilder().
			WithProxyStatsMatcher([]string{outlierDetectionRegexp}).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, targetNamespace, infrahelpers.IgnoreAlreadyExists())
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, sourceNamespace,
			infrahelpers.WithSidecarInjectionEnabled(),
			infrahelpers.IgnoreAlreadyExists())
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().
			WithNamespace(targetNamespace).
			DeployWithCleanup(t)
		require.NoError(t, err)

		err = createOutlierDetectionDestinationRule(t, r, targetNamespace, httpbinHost)
		require.NoError(t, err)

		// Pod annotation overrides with a different, unrelated pattern — NOT outlier_detection.
		curlName := "curl-override"
		err = deployCurlPod(t, r, curlName, sourceNamespace, map[string]string{
			"proxy.istio.io/config": `proxyStatsMatcher:
  inclusionRegexps:
    - ".*listener_manager.*"`,
		})
		require.NoError(t, err)

		// when: trigger ejections from the annotated pod
		for _, status := range []int{501, 502, 503, 504, 505} {
			url := fmt.Sprintf("%s/status/%d", httpbinURL, status)
			execCurl(t, r, curlName, sourceNamespace, url)
		}

		// then: the annotation replaced the global config, so outlier_detection stats are absent
		stats := getProxyStats(t, r, curlName, sourceNamespace)
		assert.NotContains(t, stats, "outlier_detection",
			"annotation should replace global proxyStatsMatcher, not extend it")

		// and: the annotated pattern IS present, confirming the annotation took effect
		assert.Contains(t, stats, "listener_manager",
			"expected listener_manager stats from pod annotation to be present")
	})

	t.Run("Outlier detection metrics are exposed on ingress gateway proxy when proxyStatsMatcher is configured via Istio CR", func(t *testing.T) {
		// Ejection state is tracked by the proxy that originates the request — when traffic
		// arrives via the ingress gateway, it is the gateway proxy (not a sidecar) that evaluates
		// outlier detection and records the ejection metrics.

		// given
		r, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureEvaluationClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().
			WithProxyStatsMatcher([]string{outlierDetectionRegexp}).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, targetNamespace, infrahelpers.IgnoreAlreadyExists())
		require.NoError(t, err)

		err = infrahelpers.CreateNamespace(t, sourceNamespace,
			infrahelpers.WithSidecarInjectionEnabled(),
			infrahelpers.IgnoreAlreadyExists())
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().
			WithNamespace(targetNamespace).
			DeployWithCleanup(t)
		require.NoError(t, err)

		err = createOutlierDetectionDestinationRule(t, r, targetNamespace, httpbinHost)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(t, "httpbin", "kyma-system", httpbinHost, httpbinHost, gatewayhelper.GatewayReference)
		require.NoError(t, err)

		// Deploy a curl pod to send requests through the ingress gateway from inside the cluster.
		curlName := "curl-ingress"
		err = deployCurlPod(t, r, curlName, sourceNamespace, nil)
		require.NoError(t, err)

		// when: send 5xx requests to the ingress gateway service; the Host header routes them
		// to httpbin via the VirtualService, causing the gateway proxy to track ejections.
		ingressURL := "http://istio-ingressgateway.istio-system.svc.cluster.local"
		for _, status := range []int{501, 502, 503, 504, 505} {
			execCurlWithHost(t, r, curlName, sourceNamespace,
				fmt.Sprintf("%s/status/%d", ingressURL, status),
				httpbinHost)
		}

		// then: the ingress gateway proxy should report ejections_active == 1
		ingressPodName, err := getIngressGatewayPodName(t, r)
		require.NoError(t, err)

		assertOutlierDetectionMetric(t, r, ingressPodName, "istio-system", 1)
	})
}

func createOutlierDetectionDestinationRule(t *testing.T, r *resources.Resources, namespace, host string) error {
	t.Helper()

	consecutive5xx := uint32(5)
	dr := &networkingv1.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1",
			Kind:       "DestinationRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "httpbin-outlier-detection",
			Namespace: namespace,
		},
		Spec: apinetworkingv1.DestinationRule{
			Host: host,
			TrafficPolicy: &apinetworkingv1.TrafficPolicy{
				OutlierDetection: &apinetworkingv1.OutlierDetection{
					Consecutive_5XxErrors: &wrappers.UInt32Value{Value: consecutive5xx},
					Interval:              &duration.Duration{Seconds: 60},
					BaseEjectionTime:      &duration.Duration{Seconds: 300},
					MaxEjectionPercent:    100,
				},
			},
		},
	}

	err := r.Create(t.Context(), dr)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create DestinationRule: %w", err)
	}

	setup.DeclareCleanup(t, func() {
		_ = r.Delete(setup.GetCleanupContext(), dr)
	})

	return nil
}

func deployCurlPodWithStatsAnnotation(t *testing.T, r *resources.Resources, name, namespace string) error {
	t.Helper()
	annotations := map[string]string{
		"proxy.istio.io/config": `proxyStatsMatcher:
  inclusionRegexps:
    - ".*outlier_detection.*"`,
	}
	return deployCurlPod(t, r, name, namespace, annotations)
}

func deployCurlPod(t *testing.T, r *resources.Resources, name, namespace string, annotations map[string]string) error {
	t.Helper()
	t.Logf("deploying curl pod %s/%s", namespace, name)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      map[string]string{"app": name},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "curl",
					Image:   "curlimages/curl:8.14.1",
					Command: []string{"/bin/sleep", "36000"},
				},
			},
		},
	}

	err := r.Create(t.Context(), pod)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create pod %s/%s: %w", namespace, name, err)
	}

	setup.DeclareCleanup(t, func() {
		t.Logf("deleting curl pod %s/%s", namespace, name)
		_ = r.Delete(setup.GetCleanupContext(), pod)
	})

	return wait.For(
		conditions.New(r).PodRunning(pod),
		wait.WithTimeout(podReadyTimeout),
		wait.WithContext(t.Context()),
	)
}

// execCurl runs a fire-and-forget curl from inside the pod.
func execCurl(t *testing.T, r *resources.Resources, podName, namespace, url string) {
	t.Helper()
	execCurlWithHost(t, r, podName, namespace, url, "")
}

// execCurlWithHost runs curl with an explicit Host header when host is non-empty.
func execCurlWithHost(t *testing.T, r *resources.Resources, podName, namespace, url, host string) {
	t.Helper()
	cmd := []string{"curl", "-s", "-o", "/dev/null", url}
	if host != "" {
		cmd = []string{"curl", "-s", "-o", "/dev/null", "-H", "Host: " + host, url}
	}
	var stdout, stderr bytes.Buffer
	err := r.ExecInPod(t.Context(), namespace, podName, "curl", cmd, &stdout, &stderr)
	if err != nil {
		t.Logf("curl exec to %s from %s/%s: %v (stderr: %s)", url, namespace, podName, err, stderr.String())
	}
}

func getProxyStats(t *testing.T, r *resources.Resources, podName, namespace string) string {
	t.Helper()
	cmd := []string{"pilot-agent", "request", "GET", "/stats/prometheus"}
	var stdout, stderr bytes.Buffer
	err := r.ExecInPod(t.Context(), namespace, podName, "istio-proxy", cmd, &stdout, &stderr)
	if err != nil {
		t.Logf("failed to get proxy stats from %s/%s: %v", namespace, podName, err)
		return ""
	}
	return stdout.String()
}

// assertOutlierDetectionMetric polls the sidecar stats of podName until the
// envoy_cluster_outlier_detection_ejections_active gauge for httpbinHost equals expectedCount.
func assertOutlierDetectionMetric(t *testing.T, r *resources.Resources, podName, namespace string, expectedCount int) {
	t.Helper()
	expectedLine := fmt.Sprintf(
		`envoy_cluster_outlier_detection_ejections_active{cluster_name="outbound|8000||%s"} %d`,
		httpbinHost, expectedCount,
	)

	err := wait.For(func(ctx context.Context) (bool, error) {
		stats := getProxyStats(t, r, podName, namespace)
		if strings.Contains(stats, expectedLine) {
			return true, nil
		}
		t.Logf("waiting for %q in %s/%s proxy stats", expectedLine, namespace, podName)
		return false, nil
	}, wait.WithTimeout(metricsWaitTimeout), wait.WithContext(t.Context()))

	require.NoError(t, err,
		"metric %q not found in %s/%s proxy stats within timeout", expectedLine, namespace, podName)
}

// getIngressGatewayPodName returns the name of the first running istio-ingressgateway pod.
func getIngressGatewayPodName(t *testing.T, r *resources.Resources) (string, error) {
	t.Helper()
	podList := &corev1.PodList{}
	err := r.List(t.Context(), podList,
		resources.WithLabelSelector("app=istio-ingressgateway"),
		resources.WithFieldSelector("metadata.namespace=istio-system"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to list ingress gateway pods: %w", err)
	}
	for _, pod := range podList.Items {
		if pod.Status.Phase == corev1.PodRunning {
			return pod.Name, nil
		}
	}
	return "", fmt.Errorf("no running istio-ingressgateway pod found in istio-system")
}
