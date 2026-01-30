package xff_header

import (
	"fmt"
	httpbinassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/httpbin"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/public_ip"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
	"github.com/stretchr/testify/require"
	"testing"
)

const defaultNamespace = "default"

func TestConfiguration(t *testing.T) {
	t.Run("X-Forward-For header contains public client IP when externalTrafficPolicy is set to Local", func(t *testing.T) {
		// given
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().
			WithGatewayExternalTrafficPolicy("Local").
			ApplyAndCleanup(t)
		require.NoError(t, err)

		istioassert.AssertReadyStatus(t, c, istioCR)

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(defaultNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"test-vs",
			defaultNamespace,
			httpbinDeployment.Host,
			httpbinDeployment.Host,
			gatewayhelper.GatewayReference,
		)
		require.NoError(t, err)

		//when
		httpClient := httphelper.NewHTTPClient(t,
			httphelper.WithHost(httpbinDeployment.Host),
		)

		//then
		clientIP, err := public_ip.FetchPublicIP(t)
		require.NoError(t, err)

		gatewayAddress, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		url := fmt.Sprintf("http://%s/get?show_env=true", gatewayAddress)

		httpbinassert.AssertHeaders(t, httpClient, url,
			httpbinassert.WithHeaderValue("X-Forwarded-For", clientIP))

	})
}
