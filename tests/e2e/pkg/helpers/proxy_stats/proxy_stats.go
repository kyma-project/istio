package proxy_stats

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/wrappers"
	apinetworkingv1 "istio.io/api/networking/v1"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

const podReadyTimeout = 2 * time.Minute

func CreateOutlierDetectionDestinationRule(t *testing.T, r *resources.Resources, namespace, host string, consecutive5xx uint32, interval, baseEjectionTime time.Duration) error {
	t.Helper()
	t.Logf("creating outlier detection DestinationRule for %s/%s", namespace, host)

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
					Interval:              &duration.Duration{Seconds: int64(interval / time.Second)},
					BaseEjectionTime:      &duration.Duration{Seconds: int64(baseEjectionTime / time.Second)},
					MaxEjectionPercent:    100,
				},
			},
		},
	}

	err := r.Create(t.Context(), dr)
	if err != nil && !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create DestinationRule: %w", err)
	}

	t.Logf("created outlier detection DestinationRule for %s/%s", namespace, host)

	setup.DeclareCleanup(t, func() {
		_ = r.Delete(setup.GetCleanupContext(), dr)
	})

	return nil
}

func DeployCurlPodWithStatsAnnotation(t *testing.T, r *resources.Resources, name, namespace string) error {
	t.Helper()
	annotations := map[string]string{
		"proxy.istio.io/config": `proxyStatsMatcher:
  inclusionRegexps:
    - ".*outlier_detection.*"`,
	}
	return DeployCurlPod(t, r, name, namespace, annotations)
}

func DeployCurlPod(t *testing.T, r *resources.Resources, name, namespace string, annotations map[string]string) error {
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

	if err := wait.For(
		conditions.New(r).PodRunning(pod),
		wait.WithTimeout(podReadyTimeout),
		wait.WithContext(t.Context()),
	); err != nil {
		return err
	}
	t.Logf("curl pod %s/%s is ready", namespace, name)
	return nil
}
func ExecCurl(t *testing.T, r *resources.Resources, podName, namespace, url string) error {
	t.Helper()
	return ExecCurlWithHost(t, r, podName, namespace, url, "")
}

func ExecCurlWithHost(t *testing.T, r *resources.Resources, podName, namespace, url, host string) error {
	t.Helper()
	cmd := []string{"curl", "-s", "-o", "/dev/null", url}
	if host != "" {
		cmd = []string{"curl", "-s", "-o", "/dev/null", "-H", "Host: " + host, url}
	}
	var stdout, stderr bytes.Buffer
	err := r.ExecInPod(t.Context(), namespace, podName, "curl", cmd, &stdout, &stderr)
	if err != nil {
		return fmt.Errorf("curl exec to %s from %s/%s failed: %w (stderr: %s)", url, namespace, podName, err, stderr.String())
	}
	return nil
}

func GetHTTPStatusCode(t *testing.T, r *resources.Resources, podName, namespace, url, host string) (string, error) {
	t.Helper()
	cmd := []string{"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", url}
	if host != "" {
		cmd = []string{"curl", "-s", "-o", "/dev/null", "-w", "%{http_code}", "-H", "Host: " + host, url}
	}
	var stdout, stderr bytes.Buffer
	err := r.ExecInPod(t.Context(), namespace, podName, "curl", cmd, &stdout, &stderr)
	if err != nil {
		return "", fmt.Errorf("curl exec to %s from %s/%s failed: %w (stderr: %s)", url, namespace, podName, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func GetProxyStats(t *testing.T, r *resources.Resources, podName, namespace string) string {
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

func GetIngressGatewayPodName(t *testing.T, r *resources.Resources) (string, error) {
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
	t.Logf("no running istio-ingressgateway pod found in istio-system")
	return "", fmt.Errorf("no running istio-ingressgateway pod found in istio-system")
}
