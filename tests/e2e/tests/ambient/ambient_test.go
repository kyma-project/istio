package ambient

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

const (
	defaultNamespace = "default"
)

func TestAmbientMode(t *testing.T) {
	c, err := client.ResourcesClient(t)
	require.NoError(t, err)

	err = infrahelpers.EnsureProductionClusterProfile(t)
	require.NoError(t, err)

	istioCR, err := modulehelpers.NewIstioCRBuilder().
		WithEnableAmbient(true).
		ApplyAndCleanup(t)
	require.NoError(t, err)

	t.Run("IstioCR is in Ready state after ambient mode is enabled", func(t *testing.T) {
		istioassert.AssertReadyStatus(t, c, istioCR)
	})

	t.Run("Ztunnel daemonset is deployed and ready when ambient mode is enabled", func(t *testing.T) {
		istioassert.AssertIstiodReady(t, c)
		istioassert.AssertIngressGatewayReady(t, c)
		istioassert.AssertCNINodeReady(t, c)
		istioassert.AssertZtunnelReady(t, c)
	})

	t.Run("Httpbin exposed with ambient mesh is accessible", func(t *testing.T) {
		err = namespace.LabelNamespaceWithAmbient(t, defaultNamespace)
		require.NoError(t, err)

		httpbinInfo, err := httpbin.NewBuilder().WithNamespace(defaultNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"httpbin-vs",
			defaultNamespace,
			httpbinInfo.Host,
			httpbinInfo.Host,
			gatewayhelper.GatewayReference,
		)
		require.NoError(t, err)

		gatewayAddress, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		hc := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("ambient-test"),
			httphelper.WithHost(httpbinInfo.Host),
		)
		url := fmt.Sprintf("http://%s/status/200", gatewayAddress)

		httpassert.AssertOKResponse(t, hc, url)
	})
}
