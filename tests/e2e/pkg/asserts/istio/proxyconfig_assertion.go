package istioassert

import (
	"strings"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/proxy_config"
	"github.com/stretchr/testify/require"
)

// AssertListenerAddressInSubnet asserts that a listener exists for the given host and port,
// and that its socket address starts with the expected subnet prefix.
func AssertListenerAddressInSubnet(t *testing.T, podName, podNamespace, serviceEntryHost string, serviceEntryPort int, expectedSubnetPrefix string) {
	t.Helper()

	listeners, err := proxy_config.GetDynamicListeners(t, podName, podNamespace)
	require.NoError(t, err, "Failed to get dynamic listeners from pod %s/%s", podNamespace, podName)

	listener := proxy_config.FindListenerByHostAndPort(listeners, serviceEntryHost, serviceEntryPort)
	require.NotNil(t, listener, "Listener for host %s and port %d not found", serviceEntryHost, serviceEntryPort)

	require.True(t,
		strings.HasPrefix(listener.SocketAddress, expectedSubnetPrefix),
		"Expected socket address %s to have prefix %s for host %s",
		listener.SocketAddress, expectedSubnetPrefix, serviceEntryHost)

	t.Logf("Listener for %s:%d has socket address %s (expected prefix: %s)", serviceEntryHost, serviceEntryPort, listener.SocketAddress, expectedSubnetPrefix)
}

// AssertListenerNotFound asserts that no listener exists for the given host and port.
func AssertListenerNotFound(t *testing.T, podName, namespace, host string, port int) {
	t.Helper()

	listeners, err := proxy_config.GetDynamicListeners(t, podName, namespace)
	require.NoError(t, err, "Failed to get dynamic listeners from pod %s/%s", namespace, podName)

	listener := proxy_config.FindListenerByHostAndPort(listeners, host, port)
	require.Nil(t, listener, "Expected no listener for host %s and port %d, but found one with address %s",
		host, port, getAddressOrEmpty(listener))

	t.Logf("Confirmed no listener exists for %s:%d", host, port)
}

// AssertListenerExists asserts that a listener exists for the given host and port.
// Returns the listener data for further assertions if needed.
func AssertListenerExists(t *testing.T, podName, namespace, host string, port int) *proxy_config.ListenerData {
	t.Helper()

	listeners, err := proxy_config.GetDynamicListeners(t, podName, namespace)
	require.NoError(t, err, "Failed to get dynamic listeners from pod %s/%s", namespace, podName)

	listener := proxy_config.FindListenerByHostAndPort(listeners, host, port)
	require.NotNil(t, listener, "Listener for host %s and port %d not found", host, port)

	t.Logf("Found listener for %s:%d with socket address %s", host, port, listener.SocketAddress)
	return listener
}

func getAddressOrEmpty(listener *proxy_config.ListenerData) string {
	if listener == nil {
		return ""
	}
	return listener.SocketAddress
}
