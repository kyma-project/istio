package proxy_stats

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	proxystatsassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/proxy_stats"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	proxystatshelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/proxy_stats"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

const (
	targetNamespace = "target"
	sourceNamespace = "source"

	outlierDetectionRegexp                  = ".*outlier_detection.*"
	outlierDetectionConsecutive5xx   uint32 = 5
	outlierDetectionInterval                = 60 * time.Second
	outlierDetectionBaseEjectionTime        = 300 * time.Second
)

func TestProxyStatsMatcher(t *testing.T) {
	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
	require.NoError(t, err)

	require.NoError(t, infrahelpers.CreateNamespace(t, targetNamespace))
	require.NoError(t, infrahelpers.CreateNamespace(t, sourceNamespace, infrahelpers.WithSidecarInjectionEnabled()))

	httpbinInfo, err := httpbin.NewBuilder().WithNamespace(targetNamespace).DeployWithCleanup(t)
	require.NoError(t, err)

	require.NoError(t, proxystatshelper.CreateOutlierDetectionDestinationRule(
		t, r, targetNamespace, httpbinInfo.Host,
		outlierDetectionConsecutive5xx, outlierDetectionInterval, outlierDetectionBaseEjectionTime,
	))

	httpbinURL := httpbinServiceURL(httpbinInfo)

	t.Run("Outlier detection metrics are exposed on all proxies when proxyStatsMatcher is configured globally via Istio CR", func(t *testing.T) {
		// given
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithProxyStatsMatcher([]string{outlierDetectionRegexp}).Update(t))

		curlName := "curl-global"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, sourceNamespace, nil))

		// when
		triggerStatusCodesFromPod(t, r, curlName, sourceNamespace, httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, sourceNamespace, fmt.Sprintf("%s/headers", httpbinURL), "", "503")
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curlName, sourceNamespace, httpbinInfo.Host, 1)
	})

	t.Run("Outlier detection metrics are not exposed when proxyStatsMatcher is not configured", func(t *testing.T) {
		// given
		require.NoError(t, modulehelpers.NewIstioCRBuilder().Update(t))

		curlName := "curl-no-stats"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, sourceNamespace, nil))

		// when
		triggerStatusCodesFromPod(t, r, curlName, sourceNamespace, httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		stats := proxystatshelper.GetProxyStats(t, r, curlName, sourceNamespace)
		require.NotContains(t, stats, "outlier_detection",
			"expected no outlier_detection metrics without proxyStatsMatcher")
	})

	t.Run("Per-workload annotation replaces global proxyStatsMatcher instead of extending it", func(t *testing.T) {
		// given
		// The docs state that proxy.istio.io/config replaces the global proxyStatsMatcher for
		// that workload. This test verifies the replacement semantics: a pod annotated with
		// pattern A does not inherit pattern B from the global Istio CR.
		require.NoError(t, modulehelpers.NewIstioCRBuilder().Update(t))

		// Pod annotation overrides with a different, unrelated pattern — NOT outlier_detection.
		curlName := "curl-override"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, sourceNamespace, map[string]string{
			"proxy.istio.io/config": `proxyStatsMatcher:
  inclusionRegexps:
    - ".*listener_manager.*"`,
		}))

		// when
		triggerStatusCodesFromPod(t, r, curlName, sourceNamespace, httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, sourceNamespace, fmt.Sprintf("%s/headers", httpbinURL), "", "503")

		stats := proxystatshelper.GetProxyStats(t, r, curlName, sourceNamespace)
		require.NotContains(t, stats, "outlier_detection",
			"annotation should replace global proxyStatsMatcher, not extend it")
		require.Contains(t, stats, "listener_manager",
			"expected listener_manager stats from pod annotation to be present")
	})

	t.Run("Outlier detection metrics are exposed on source proxy when proxyStatsMatcher is configured via pod annotation", func(t *testing.T) {
		// given
		require.NoError(t, modulehelpers.NewIstioCRBuilder().Update(t))

		// Deploy two curl pods with outlier_detection metrics enabled via annotation.
		// Each pod tracks ejection state independently.
		curl1Name := "curl-1"
		curl2Name := "curl-2"

		require.NoError(t, proxystatshelper.DeployCurlPodWithStatsAnnotation(t, r, curl1Name, sourceNamespace))
		require.NoError(t, proxystatshelper.DeployCurlPodWithStatsAnnotation(t, r, curl2Name, sourceNamespace))

		// when
		triggerStatusCodesFromPod(t, r, curl1Name, sourceNamespace, httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curl1Name, sourceNamespace, fmt.Sprintf("%s/headers", httpbinURL), "", "503")
		proxystatsassert.AssertHTTPStatusCode(t, r, curl2Name, sourceNamespace, fmt.Sprintf("%s/headers", httpbinURL), "", "200")

		proxystatsassert.AssertOutlierDetectionMetric(t, r, curl1Name, sourceNamespace, httpbinInfo.Host, 1)
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curl2Name, sourceNamespace, httpbinInfo.Host, 0)
	})

	t.Run("Outlier detection metrics are exposed on ingress gateway proxy when proxyStatsMatcher is configured via Istio CR", func(t *testing.T) {
		// given
		// Ejection state is tracked by the proxy that originates the request — when traffic
		// arrives via the ingress gateway, it is the gateway proxy (not a sidecar) that evaluates
		// outlier detection and records the ejection metrics.
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithProxyStatsMatcher([]string{outlierDetectionRegexp}).Update(t))

		require.NoError(t, gatewayhelper.CreateHTTPGateway(t))
		require.NoError(t, virtualservice.CreateVirtualService(t, "httpbin", "kyma-system", httpbinInfo.Host, httpbinInfo.Host, gatewayhelper.GatewayReference))

		// deploy a curl pod to send requests through the ingress gateway from inside the cluster
		curlName := "curl-ingress"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, sourceNamespace, nil))

		// when
		ingressURL := "http://istio-ingressgateway.istio-system.svc.cluster.local"
		triggerStatusCodesViaIngress(t, r, curlName, sourceNamespace, ingressURL, httpbinInfo.Host, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, sourceNamespace, fmt.Sprintf("%s/headers", ingressURL), httpbinInfo.Host, "503")

		ingressPodName, err := proxystatshelper.GetIngressGatewayPodName(t, r)
		require.NoError(t, err)

		// ejection is tracked by the ingress gateway proxy, not the curl sidecar
		proxystatsassert.AssertOutlierDetectionMetric(t, r, ingressPodName, "istio-system", httpbinInfo.Host, 1)
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curlName, sourceNamespace, httpbinInfo.Host, 0)
	})
}

func httpbinServiceURL(httpbinInfo *httpbin.DeploymentInfo) string {
	return fmt.Sprintf("http://%s:%d", httpbinInfo.Host, httpbinInfo.Port)
}

func triggerStatusCodesFromPod(t *testing.T, r *resources.Resources, podName, namespace, targetURL string, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		require.NoError(t, proxystatshelper.ExecCurl(t, r, podName, namespace, fmt.Sprintf("%s/status/500", targetURL)))
	}
}

func triggerStatusCodesViaIngress(t *testing.T, r *resources.Resources, podName, namespace, ingressURL, host string, count int) {
	t.Helper()

	for i := 0; i < count; i++ {
		require.NoError(t, proxystatshelper.ExecCurlWithHost(t, r, podName, namespace, fmt.Sprintf("%s/status/500", ingressURL), host))
	}
}
