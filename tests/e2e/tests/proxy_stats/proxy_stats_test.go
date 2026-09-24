package proxy_stats

import (
	"fmt"
	"hash/fnv"
	"strings"
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

	t.Run("Outlier detection metrics are exposed on all proxies when proxyStatsMatcher is configured globally via Istio CR", func(t *testing.T) {
		// given
		_, err := modulehelpers.NewIstioCRBuilder().WithProxyStatsMatcher([]string{outlierDetectionRegexp}).ApplyAndCleanup(t)
		require.NoError(t, err)

		scenario := setupProxyStatsScenario(t, r)

		curlName := "curl-global"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, scenario.sourceNamespace, nil))

		// when
		triggerStatusCodesFromPod(t, r, curlName, scenario.sourceNamespace, scenario.httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, scenario.sourceNamespace, fmt.Sprintf("%s/headers", scenario.httpbinURL), "", "503")
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curlName, scenario.sourceNamespace, scenario.httpbinInfo.Host, 1)
	})

	t.Run("Outlier detection metrics are not exposed when proxyStatsMatcher is not configured", func(t *testing.T) {
		// given
		_, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		scenario := setupProxyStatsScenario(t, r)

		curlName := "curl-no-stats"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, scenario.sourceNamespace, nil))

		// when
		triggerStatusCodesFromPod(t, r, curlName, scenario.sourceNamespace, scenario.httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertProxyStatsAbsent(t, r, curlName, scenario.sourceNamespace, "outlier_detection")
	})

	t.Run("Per-workload annotation replaces global proxyStatsMatcher instead of extending it", func(t *testing.T) {
		// given
		_, err := modulehelpers.NewIstioCRBuilder().WithProxyStatsMatcher([]string{outlierDetectionRegexp}).ApplyAndCleanup(t)
		require.NoError(t, err)

		scenario := setupProxyStatsScenario(t, r)

		// Pod annotation overrides with a different pattern; outlier_detection must not appear.
		curlName := "curl-override"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, scenario.sourceNamespace, map[string]string{
			"proxy.istio.io/config": `proxyStatsMatcher:
 inclusionRegexps:
   - ".*listener_manager.*"`,
		}))

		// when
		triggerStatusCodesFromPod(t, r, curlName, scenario.sourceNamespace, scenario.httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, scenario.sourceNamespace, fmt.Sprintf("%s/headers", scenario.httpbinURL), "", "503")
		proxystatsassert.AssertProxyStatsAbsent(t, r, curlName, scenario.sourceNamespace, "outlier_detection")
		proxystatsassert.AssertProxyStatsPresent(t, r, curlName, scenario.sourceNamespace, "listener_manager")
	})

	t.Run("Outlier detection metrics are exposed on source proxy when proxyStatsMatcher is configured via pod annotation", func(t *testing.T) {
		// given
		_, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		scenario := setupProxyStatsScenario(t, r)

		// Two pods with the annotation; only the one that sends 5xx traffic should record an ejection.
		curl1Name := "curl-1"
		curl2Name := "curl-2"

		require.NoError(t, proxystatshelper.DeployCurlPodWithStatsAnnotation(t, r, curl1Name, scenario.sourceNamespace))
		require.NoError(t, proxystatshelper.DeployCurlPodWithStatsAnnotation(t, r, curl2Name, scenario.sourceNamespace))

		// when
		triggerStatusCodesFromPod(t, r, curl1Name, scenario.sourceNamespace, scenario.httpbinURL, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curl1Name, scenario.sourceNamespace, fmt.Sprintf("%s/headers", scenario.httpbinURL), "", "503")
		proxystatsassert.AssertHTTPStatusCode(t, r, curl2Name, scenario.sourceNamespace, fmt.Sprintf("%s/headers", scenario.httpbinURL), "", "200")

		proxystatsassert.AssertOutlierDetectionMetric(t, r, curl1Name, scenario.sourceNamespace, scenario.httpbinInfo.Host, 1)
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curl2Name, scenario.sourceNamespace, scenario.httpbinInfo.Host, 0)
	})

	t.Run("Outlier detection metrics are exposed on ingress gateway proxy with single replica when proxyStatsMatcher is configured via Istio CR", func(t *testing.T) {
		// given
		_, err := modulehelpers.NewIstioCRBuilder().
			WithProxyStatsMatcher([]string{outlierDetectionRegexp}).
			WithIngressGatewayHPA(1, 1).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		scenario := setupProxyStatsScenario(t, r)

		// Single replica so all requests land on the same pod, making ejection state deterministic.
		require.NoError(t, waitForIngressGatewayReplicas(t, r, 1))
		require.NoError(t, gatewayhelper.CreateHTTPGateway(t))
		require.NoError(t, virtualservice.CreateVirtualService(t, "httpbin", "kyma-system", scenario.httpbinInfo.Host, scenario.httpbinInfo.Host, gatewayhelper.GatewayReference))

		curlName := "curl-ingress-single"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, scenario.sourceNamespace, nil))

		// when
		ingressURL := "http://istio-ingressgateway.istio-system.svc.cluster.local"
		triggerStatusCodesViaIngress(t, r, curlName, scenario.sourceNamespace, ingressURL, scenario.httpbinInfo.Host, int(outlierDetectionConsecutive5xx))

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, scenario.sourceNamespace, fmt.Sprintf("%s/headers", ingressURL), scenario.httpbinInfo.Host, "503")

		ingressPodName, err := proxystatshelper.GetIngressGatewayPodName(t, r)
		require.NoError(t, err)

		// Ejection is tracked on the ingress gateway pod that processed the upstream requests.
		proxystatsassert.AssertOutlierDetectionMetric(t, r, ingressPodName, "istio-system", scenario.httpbinInfo.Host, 1)
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curlName, scenario.sourceNamespace, scenario.httpbinInfo.Host, 0)
	})

	t.Run("Outlier detection metrics are exposed on ingress gateway proxy with multiple replicas when proxyStatsMatcher is configured via Istio CR", func(t *testing.T) {
		const numReplicas int32 = 3

		// given
		_, err := modulehelpers.NewIstioCRBuilder().
			WithProxyStatsMatcher([]string{outlierDetectionRegexp, upstreamRequestRegexp}).
			WithIngressGatewayHPA(numReplicas, numReplicas).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		scenario := setupProxyStatsScenario(t, r)

		// Sends numReplicas*threshold requests so at least one pod crosses the ejection threshold.
		require.NoError(t, waitForIngressGatewayReplicas(t, r, numReplicas))
		require.NoError(t, gatewayhelper.CreateHTTPGateway(t))
		require.NoError(t, virtualservice.CreateVirtualService(t, "httpbin", "kyma-system", scenario.httpbinInfo.Host, scenario.httpbinInfo.Host, gatewayhelper.GatewayReference))

		curlName := "curl-ingress"
		require.NoError(t, proxystatshelper.DeployCurlPod(t, r, curlName, scenario.sourceNamespace, nil))

		// when
		ingressURL := "http://istio-ingressgateway.istio-system.svc.cluster.local"
		requestCount := int(numReplicas) * int(outlierDetectionConsecutive5xx)
		for range requestCount {
			_, err := proxystatshelper.GetHTTPStatusCode(t, r, curlName, scenario.sourceNamespace, fmt.Sprintf("%s/status/500", ingressURL), scenario.httpbinInfo.Host)
			require.NoError(t, err)
		}

		// then
		proxystatsassert.AssertHTTPStatusCode(t, r, curlName, scenario.sourceNamespace, fmt.Sprintf("%s/headers", ingressURL), scenario.httpbinInfo.Host, "503")
		proxystatsassert.AssertAllIngressGatewayPodsHaveEjectionMetric(t, r, scenario.httpbinInfo.Host)
		proxystatsassert.AssertEjectionStateMatchesUpstream5xx(t, r, scenario.httpbinInfo.Host, int(outlierDetectionConsecutive5xx))

		// Ejection is tracked by the ingress gateway proxy, not the curl sidecar.
		proxystatsassert.AssertOutlierDetectionMetric(t, r, curlName, scenario.sourceNamespace, scenario.httpbinInfo.Host, 0)
	})
}

type proxyStatsScenario struct {
	targetNamespace string
	sourceNamespace string
	httpbinInfo     *httpbin.DeploymentInfo
	httpbinURL      string
}

func httpbinServiceURL(httpbinInfo *httpbin.DeploymentInfo) string {
	return fmt.Sprintf("http://%s:%d", httpbinInfo.Host, httpbinInfo.Port)
}

func setupProxyStatsScenario(t *testing.T, r *resources.Resources) proxyStatsScenario {
	t.Helper()

	targetNS := uniqueTestNamespace(targetNamespace, t.Name())
	sourceNS := uniqueTestNamespace(sourceNamespace, t.Name())

	require.NoError(t, infrahelpers.CreateNamespace(t, targetNS))
	require.NoError(t, infrahelpers.CreateNamespace(t, sourceNS, infrahelpers.WithSidecarInjectionEnabled()))

	httpbinInfo, err := httpbin.NewBuilder().WithNamespace(targetNS).DeployWithCleanup(t)
	require.NoError(t, err)

	require.NoError(t, proxystatshelper.CreateOutlierDetectionDestinationRule(
		t, r, targetNS, httpbinInfo.Host,
		outlierDetectionConsecutive5xx, outlierDetectionInterval, outlierDetectionBaseEjectionTime,
	))

	return proxyStatsScenario{
		targetNamespace: targetNS,
		sourceNamespace: sourceNS,
		httpbinInfo:     httpbinInfo,
		httpbinURL:      httpbinServiceURL(httpbinInfo),
	}
}

func uniqueTestNamespace(prefix, testName string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(testName))
	hash := fmt.Sprintf("%08x", h.Sum32())

	sanitized := strings.ToLower(testName)
	var b strings.Builder
	b.Grow(len(sanitized))
	lastDash := false
	for _, r := range sanitized {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	name := strings.Trim(b.String(), "-")
	if name == "" {
		name = prefix
	}

	maxBaseLen := 63 - len(prefix) - len(hash) - 2
	if maxBaseLen < 1 {
		maxBaseLen = 1
	}
	if len(name) > maxBaseLen {
		name = name[:maxBaseLen]
		name = strings.Trim(name, "-")
		if name == "" {
			name = prefix
		}
	}

	return fmt.Sprintf("%s-%s-%s", prefix, name, hash)
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
