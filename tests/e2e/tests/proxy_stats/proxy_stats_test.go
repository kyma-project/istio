package proxy_stats

import (
	"fmt"
	"testing"
	"time"

	proxystatsassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/proxy_stats"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	proxystatshelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/proxy_stats"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
)

const (
	targetNamespace = "target"
	sourceNamespace = "source"

	outlierDetectionRegexp                  = ".*outlier_detection.*"
	upstreamRequestRegexp                   = ".*upstream_rq.*"
	outlierDetectionConsecutive5xx   uint32 = 5
	outlierDetectionInterval                = 60 * time.Second
	outlierDetectionBaseEjectionTime        = 300 * time.Second
)

func TestProxyStatsMatcher(t *testing.T) {
	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	// Create the Istio CR once; each subtest updates it and reverts on cleanup.
	_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
	require.NoError(t, err)

	// Shared scenario: namespaces, httpbin, and destination rule are created once
	// and reused across all subtests. Individual curl pods are created per subtest.
	require.NoError(t, infrahelpers.CreateNamespace(t, targetNamespace))
	require.NoError(t, infrahelpers.CreateNamespace(t, sourceNamespace, infrahelpers.WithSidecarInjectionEnabled()))

	httpbinInfo, err := httpbin.NewBuilder().WithNamespace(targetNamespace).DeployWithCleanup(t)
	require.NoError(t, err)

	require.NoError(t, proxystatshelper.CreateOutlierDetectionDestinationRule(
		t, r, targetNamespace, httpbinInfo.Host,
		outlierDetectionConsecutive5xx, outlierDetectionInterval, outlierDetectionBaseEjectionTime,
	))

	httpbinURL := httpbinServiceURL(httpbinInfo)

	require.NoError(t, gatewayhelper.CreateHTTPGateway(t))
	require.NoError(t, virtualservice.CreateVirtualService(t, "httpbin", "kyma-system", httpbinInfo.Host, httpbinInfo.Host, gatewayhelper.GatewayReference))

	t.Run("Outlier detection metrics are exposed on all proxies when proxyStatsMatcher is configured globally via Istio CR", func(t *testing.T) {
		// given
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithProxyStatsMatcher([]string{outlierDetectionRegexp}).UpdateAndRevert(t))

		curlName := "curl-global"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, sourceNamespace, nil))

		// when
		triggerStatusCodesFromPod(t, r, curlName, sourceNamespace, httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, sourceNamespace, fmt.Sprintf("%s/headers", httpbinURL), "", "503")
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curlName, sourceNamespace, httpbinInfo.Host, 1)
	})

	t.Run("Outlier detection metrics are not exposed when proxyStatsMatcher is not configured", func(t *testing.T) {
		// given — baseline has no proxyStatsMatcher; UpdateAndRevert guards against state left by a prior subtest.
		require.NoError(t, modulehelpers.NewIstioCRBuilder().UpdateAndRevert(t))

		curlName := "curl-no-stats"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, sourceNamespace, nil))

		// when
		triggerStatusCodesFromPod(t, r, curlName, sourceNamespace, httpbinURL, int(outlierDetectionConsecutive5xx))

		// then — ejection fires (503), but the metric must be absent because no matcher is configured.
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, sourceNamespace, fmt.Sprintf("%s/headers", httpbinURL), "", "503")
		proxystatsassert.AssertProxyStatsAbsent(t, r, curlName, sourceNamespace, "outlier_detection")
	})

	t.Run("Per-workload annotation replaces global proxyStatsMatcher instead of extending it", func(t *testing.T) {
		// given
		require.NoError(t, modulehelpers.NewIstioCRBuilder().WithProxyStatsMatcher([]string{outlierDetectionRegexp}).UpdateAndRevert(t))

		// Pod annotation overrides with a different pattern; outlier_detection must not appear.
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
		proxystatsassert.AssertProxyStatsAbsent(t, r, curlName, sourceNamespace, "outlier_detection")
		proxystatsassert.AssertProxyStatsPresent(t, r, curlName, sourceNamespace, "listener_manager")
	})

	t.Run("Outlier detection metrics are exposed on source proxy when proxyStatsMatcher is configured via pod annotation", func(t *testing.T) {
		// given — no global proxyStatsMatcher; annotation on the pod provides the config.
		require.NoError(t, modulehelpers.NewIstioCRBuilder().UpdateAndRevert(t))

		// Two pods with the annotation; only the one that sends 5xx traffic should record an ejection.
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

	t.Run("Outlier detection metrics are exposed on ingress gateway proxy with single replica when proxyStatsMatcher is configured via Istio CR", func(t *testing.T) {
		// given
		require.NoError(t, modulehelpers.NewIstioCRBuilder().
			WithProxyStatsMatcher([]string{outlierDetectionRegexp}).
			WithIngressGatewayHPA(1, 1).
			UpdateAndRevert(t))

		// Single replica so all requests land on the same pod, making ejection state deterministic.
		require.NoError(t, waitForIngressGatewayReplicas(t, r, 1))

		curlName := "curl-ingress-single"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, sourceNamespace, nil))

		// when
		ingressURL := "http://istio-ingressgateway.istio-system.svc.cluster.local"
		triggerStatusCodesViaIngress(t, r, curlName, sourceNamespace, ingressURL, httpbinInfo.Host, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, sourceNamespace, fmt.Sprintf("%s/headers", ingressURL), httpbinInfo.Host, "503")

		ingressPodName, err := proxystatshelper.GetIngressGatewayPodName(t, r)
		require.NoError(t, err)

		// Ejection is tracked on the ingress gateway pod that processed the upstream requests.
		proxystatsassert.AssertOutlierDetectionMetric(t, r, ingressPodName, "istio-system", httpbinInfo.Host, 1)
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curlName, sourceNamespace, httpbinInfo.Host, 0)
	})

	t.Run("Outlier detection metrics are exposed on ingress gateway proxy with multiple replicas when proxyStatsMatcher is configured via Istio CR", func(t *testing.T) {
		const numReplicas int32 = 3

		// given
		require.NoError(t, modulehelpers.NewIstioCRBuilder().
			WithProxyStatsMatcher([]string{outlierDetectionRegexp, upstreamRequestRegexp}).
			WithIngressGatewayHPA(numReplicas, numReplicas).
			UpdateAndRevert(t))

		// Sends numReplicas*threshold requests so at least one pod crosses the ejection threshold.
		require.NoError(t, waitForIngressGatewayReplicas(t, r, numReplicas))

		curlName := "curl-ingress"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, sourceNamespace, nil))

		// when
		ingressURL := "http://istio-ingressgateway.istio-system.svc.cluster.local"
		requestCount := int(numReplicas) * int(outlierDetectionConsecutive5xx)
		for range requestCount {
			_, err := proxystatshelper.GetHTTPStatusCode(t, r, curlName, sourceNamespace, fmt.Sprintf("%s/status/500", ingressURL), httpbinInfo.Host)
			require.NoError(t, err)
		}

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, sourceNamespace, fmt.Sprintf("%s/headers", ingressURL), httpbinInfo.Host, "503")
		proxystatsassert.AssertAllIngressGatewayPodsHaveEjectionMetric(t, r, httpbinInfo.Host)
		proxystatsassert.AssertEjectionStateMatchesUpstream5xxMetric(t, r, httpbinInfo.Host, int(outlierDetectionConsecutive5xx))

		// At least one pod must have crossed the ejection threshold given numReplicas*threshold requests.
		proxystatsassert.AssertAtLeastOneIngressGatewayPodHasEjection(t, r, httpbinInfo.Host)

		// Ejection is tracked by the ingress gateway proxy, not the curl sidecar.
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curlName, sourceNamespace, httpbinInfo.Host, 0)
	})
}

func httpbinServiceURL(httpbinInfo *httpbin.DeploymentInfo) string {
	return fmt.Sprintf("http://%s:%d", httpbinInfo.Host, httpbinInfo.Port)
}

func triggerStatusCodesFromPod(t *testing.T, r *resources.Resources, podName, namespace, targetURL string, count int) {
	t.Helper()

	for range count {
		require.NoError(t, proxystatshelper.ExecCurl(t, r, podName, namespace, fmt.Sprintf("%s/status/500", targetURL)))
	}
}

func triggerStatusCodesViaIngress(t *testing.T, r *resources.Resources, podName, namespace, ingressURL, host string, count int) {
	t.Helper()

	for range count {
		require.NoError(t, proxystatshelper.ExecCurlWithHost(t, r, podName, namespace, fmt.Sprintf("%s/status/500", ingressURL), host))
	}
}

func waitForIngressGatewayReplicas(t *testing.T, r *resources.Resources, expectedReplicas int32) error {
	return modulehelpers.WaitForIngressGatewayReplicas(t.Context(), t, r, expectedReplicas)
}
