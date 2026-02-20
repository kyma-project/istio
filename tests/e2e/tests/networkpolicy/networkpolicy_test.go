package networkpolicy_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	networkpolicyassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/networkpolicy"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/egressgateway"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpincluster"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	nphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/networkpolicy"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/testsetup"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

func TestNetworkPoliciesNotCreatedByDefault(t *testing.T) {
	_, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	networkpolicyassert.AssertAllModuleNetworkPoliciesNotExist(t, r)
}

func TestNetworkPoliciesCreatedWhenEnabled(t *testing.T) {
	_, err := modulehelpers.NewIstioCRBuilder().
		WithEnableModuleNetworkPolicies(true).
		ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	networkpolicyassert.AssertAllNetworkPoliciesValid(t, r)
}

func TestNetworkPoliciesDeletedWhenDisabled(t *testing.T) {
	builder := modulehelpers.NewIstioCRBuilder().WithEnableModuleNetworkPolicies(true)
	_, err := builder.ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	networkpolicyassert.AssertAllModuleNetworkPoliciesExist(t, r)

	err = builder.WithEnableModuleNetworkPolicies(false).Update(t)
	require.NoError(t, err)

	networkpolicyassert.AssertAllModuleNetworkPoliciesNotExist(t, r)
}

func TestWorkloadWithDefaultDenyPolicy(t *testing.T) {
	_, err := modulehelpers.NewIstioCRBuilder().
		WithEnableModuleNetworkPolicies(true).
		ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	_, testNs, err := testsetup.CreateNamespaceWithRandomID(t,
		testsetup.WithPrefix("deny-all-test"),
		testsetup.WithSidecarInjectionEnabled())
	require.NoError(t, err)

	_, err = nphelper.NewDefaultDenyNetworkPolicy(testNs).DeployWithCleanup(t, r)
	require.NoError(t, err)

	httpbinLabels := map[string]string{
		"app": "httpbin",
		"networking.kyma-project.io/from-ingressgateway": "allowed",
	}

	_, err = nphelper.NewAllowFromIngressGatewayNetworkPolicy(testNs, httpbinLabels).
		DeployWithCleanup(t, r)
	require.NoError(t, err)

	_, err = nphelper.NewAllowToIstioSystemNetworkPolicy(testNs, httpbinLabels).
		DeployWithCleanup(t, r)
	require.NoError(t, err)

	deployment, err := httpbin.NewBuilder().
		WithNamespace(testNs).
		WithLabels(httpbinLabels).
		DeployWithCleanup(t)
	require.NoError(t, err)

	err = gatewayhelper.CreateHTTPGateway(t)
	require.NoError(t, err)

	err = virtual_service.CreateVirtualService(t, "httpbin-vs", testNs, deployment.Host, deployment.Host, gatewayhelper.GatewayReference)
	require.NoError(t, err)

	ingressAddr, err := load_balancer.GetLoadBalancerIP(t.Context(), r.GetControllerRuntimeClient())
	require.NoError(t, err)

	httpClient := httphelper.NewHTTPClient(t, httphelper.WithHost(deployment.Host))
	url := fmt.Sprintf("http://%s/headers", ingressAddr)
	httpassert.AssertOKResponse(t, httpClient, url)
}

func TestIngressGatewayEgressBlockedWithoutLabel(t *testing.T) {
	_, err := modulehelpers.NewIstioCRBuilder().
		WithEnableModuleNetworkPolicies(true).
		ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	_, testNs, err := testsetup.CreateNamespaceWithRandomID(t,
		testsetup.WithPrefix("no-label-test"),
		testsetup.WithSidecarInjectionEnabled())
	require.NoError(t, err)

	deployment, err := httpbin.NewBuilder().
		WithNamespace(testNs).
		DeployWithCleanup(t)
	require.NoError(t, err)

	err = gatewayhelper.CreateHTTPGateway(t)
	require.NoError(t, err)

	err = virtual_service.CreateVirtualService(t, "httpbin-vs", testNs, deployment.Host, deployment.Host, gatewayhelper.GatewayReference)
	require.NoError(t, err)

	ingressAddr, err := load_balancer.GetLoadBalancerIP(t.Context(), r.GetControllerRuntimeClient())
	require.NoError(t, err)

	// Traffic should fail because httpbin doesn't have the required label
	// NetworkPolicy blocks at network level - expect connection error (timeout/reset), not HTTP error
	httpClient := httphelper.NewHTTPClient(t,
		httphelper.WithHost(deployment.Host),
		httphelper.WithTimeout(5*time.Second))

	url := fmt.Sprintf("http://%s/headers", ingressAddr)
	httpassert.AssertConnectionError(t, httpClient, url)
}

func TestEgressGatewayTraffic(t *testing.T) {
	enabled := true
	_, err := modulehelpers.NewIstioCRBuilder().
		WithEnableModuleNetworkPolicies(true).
		WithEgressGateway(&v1alpha2.EgressGateway{
			Enabled: &enabled,
		}).
		ApplyAndCleanup(t)
	require.NoError(t, err)

	r, err := client.ResourcesClient(t)
	require.NoError(t, err)

	networkpolicyassert.AssertEgressGatewayNetworkPolicy(t, r)

	_, testNs, err := testsetup.CreateNamespaceWithRandomID(t,
		testsetup.WithPrefix("egress-gw-test"),
		testsetup.WithSidecarInjectionEnabled())
	require.NoError(t, err)

	curlLabels := map[string]string{
		"app": "curl",
		"networking.kyma-project.io/to-egressgateway": "allowed",
	}

	err = egressgateway.SetupEgressGatewayResources(t, r, testNs, "httpbin.org")
	require.NoError(t, err)

	externalURL := "https://httpbin.org/headers"
	stdout, _, err := httpincluster.RunRequestFromInsideClusterWithLabels(t, testNs, externalURL, curlLabels)
	require.NoError(t, err)
	require.Contains(t, stdout, "Host", "Request through egress gateway should succeed")
}

func TestIstiodDNSAccess(t *testing.T) {
	_, err := modulehelpers.NewIstioCRBuilder().
		WithEnableModuleNetworkPolicies(true).
		ApplyAndCleanup(t)
	require.NoError(t, err)

	// Test DNS access by making a request to a service using its DNS name from istio-system namespace
	// This validates that DNS egress is allowed by NetworkPolicy for istiod labeled pods
	// We deploy a test pod with the same label selector as istiod (istio: pilot) to verify DNS works
	stdout, stderr, err := httpincluster.RunRequestFromInsideClusterWithLabels(
		t,
		"istio-system",
		"https://kubernetes.default.svc/livez",
		map[string]string{
			"istio": "pilot", // Same label as istiod to match the NetworkPolicy
		},
	)
	t.Logf("DNS test stdout: %s", stdout)
	t.Logf("DNS test stderr: %s", stderr)

	// Check that we don't have DNS-related errors - these would indicate NetworkPolicy blocking DNS
	require.NotContains(t, stderr, "Could not resolve host", "DNS resolution should work")
	require.NotContains(t, stderr, "Couldn't resolve host", "DNS resolution should work")
	require.NotContains(t, stderr, "Name or service not known", "DNS resolution should work")

	// If we received an HTTP response (even 401), it proves DNS resolution worked.
	// The curl command uses --fail-with-body which returns error on HTTP errors,
	// but getting any HTTP response means DNS resolved and connection was established.
	if err != nil {
		// Check if we got an HTTP response - this proves DNS worked
		require.True(t, strings.Contains(stdout, "HTTP/") || strings.Contains(stderr, "401"),
			"Expected HTTP response (proves DNS worked), got: stdout=%s, stderr=%s", stdout, stderr)
	}
}
