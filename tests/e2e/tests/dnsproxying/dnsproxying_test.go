package dnsproxying_test

import (
	"testing"

	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	serviceentry "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/service_entry"
	"github.com/stretchr/testify/require"
)

const (
	testNamespace      = "test"
	external1Namespace = "external1"
	external2Namespace = "external2"
	testServiceEntry1  = "test-service-entry-1"
	testServiceEntry2  = "test-service-entry-2"
	portResolution     = "DNS"
	portProtocol       = "TCP"
	portNumber         = 9000

	tcpEchoExternal1Host = "tcp-echo.external-1.svc.cluster.local"
	tcpEchoExternal2Host = "tcp-echo.external-2.svc.cluster.local"

	// DNS proxy uses 240.240.0.0/16 subnet for auto-allocated IPs when DNS proxying is enabled and ServiceEntry exist
	dnsProxySubnetPrefix = "240.240"
	// When DNS proxy is disabled and ServiceEntry exist, listeners bind to 0.0.0.0
	wildcardAddress = "0.0.0.0"
)

// TestDNSProxying tests the behavior of Istio when DNS proxying is enabled or disabled, by verifying the addresses of TCP listeners created for ServiceEntries with DNS resolution of TCP service.
// - When DNS proxying is disabled and ServiceEntry exist, listeners should have wildcard address (0.0.0.0).
// - When DNS proxying is enabled and ServiceEntry exist, listeners should have auto-allocated address IP (240.240.0.0/16).
// - When DNS proxying enabled or disabled and no ServiceEntry exist for service, listeners should bind to cluster-IP of service.
func TestDNSProxying(t *testing.T) {
	t.Run("DNS Proxying enabled - TCP listeners for ServiceEntries with DNS resolution are created with auto-allocated IPs from range 240.240.0.0/16 for both ServiceEntries", func(t *testing.T) {
		// given
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().
			WithEnableDNSProxying(true).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		istioassert.AssertReadyStatus(t, c, istioCR)

		setupTestEnvironment(t)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(testNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinPodList, err := httpbin.GetHttpbinPods(t, httpbinDeployment.WorkloadSelector)
		require.NoError(t, err)
		require.NotEmpty(t, httpbinPodList.Items, "Expected at least one httpbin pod")

		podName := httpbinPodList.Items[0].Name

		// then - both listeners should have auto-allocated IPs from DNS proxy subnet
		istioassert.AssertListenerAddressInSubnet(t, podName, testNamespace, tcpEchoExternal1Host, portNumber, dnsProxySubnetPrefix)
		istioassert.AssertListenerAddressInSubnet(t, podName, testNamespace, tcpEchoExternal2Host, portNumber, dnsProxySubnetPrefix)
	})

	t.Run("DNS Proxying disabled - only one TCP listener created with wildcard address (0.0.0.0), second ServiceEntry is not recognized by mesh", func(t *testing.T) {
		// given
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().
			WithEnableDNSProxying(false).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		istioassert.AssertReadyStatus(t, c, istioCR)

		setupTestEnvironment(t)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(testNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinPodList, err := httpbin.GetHttpbinPods(t, httpbinDeployment.WorkloadSelector)
		require.NoError(t, err)
		require.NotEmpty(t, httpbinPodList.Items, "Expected at least one httpbin pod")

		podName := httpbinPodList.Items[0].Name

		// then - first listener uses wildcard address, second listener is not created due to port conflict
		istioassert.AssertListenerAddressInSubnet(t, podName, testNamespace, tcpEchoExternal1Host, portNumber, wildcardAddress)
		istioassert.AssertListenerNotFound(t, podName, testNamespace, tcpEchoExternal2Host, portNumber)
	})
}

// setupTestEnvironment creates all required namespaces and service entries for tests.
func setupTestEnvironment(t *testing.T) {
	t.Helper()

	// Create namespaces
	err := infrastructure.CreateNamespace(t, testNamespace, infrastructure.WithSidecarInjectionEnabled())
	require.NoError(t, err)

	err = infrastructure.CreateNamespace(t, external1Namespace, infrastructure.WithSidecarInjectionDisabled())
	require.NoError(t, err)

	err = infrastructure.CreateNamespace(t, external2Namespace, infrastructure.WithSidecarInjectionDisabled())
	require.NoError(t, err)

	// Create service entries
	err = serviceentry.CreateServiceEntry(t, testServiceEntry1, "default", tcpEchoExternal1Host, portProtocol, portResolution, portNumber)
	require.NoError(t, err)

	err = serviceentry.CreateServiceEntry(t, testServiceEntry2, "default", tcpEchoExternal2Host, portProtocol, portResolution, portNumber)
	require.NoError(t, err)
}
