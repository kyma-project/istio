package gateway_api_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/crds"
	gatewayapihelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway_api"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
)

const (
	defaultNamespace = "default"
	gatewayName      = "httpbin-gateway"
	httpRouteName    = "httpbin-route"
)

func TestGatewayAPI(t *testing.T) {
	t.Run("Gateway API CRDs are installed by Istio module", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)


		_, err = modulehelpers.NewIstioCRBuilder().WithEnableGatewayAPI(true).ApplyAndCleanup(t)
		require.NoError(t, err)

		err = crds.AssertGatewayAPICRDsPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err, "Gateway API CRDs should be present after Istio module installation")
	})

	t.Run("Httpbin is accessible through Gateway API HTTPRoute", func(t *testing.T) {

		_, err := modulehelpers.NewIstioCRBuilder().WithEnableGatewayAPI(true).ApplyAndCleanup(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		// Deploy httpbin with sidecar injection
		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(defaultNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		// Wait for the Istio GatewayClass to be accepted before creating Gateway resources
		err = gatewayapihelper.WaitForGatewayClassReady(t)
		require.NoError(t, err, "Istio GatewayClass should be accepted")

		// Create Gateway API Gateway
		err = gatewayapihelper.CreateGateway(t, gatewayName, defaultNamespace)
		require.NoError(t, err)

		// Create HTTPRoute pointing at httpbin
		err = gatewayapihelper.CreateHTTPRoute(t, httpRouteName, defaultNamespace, gatewayName, httpbinDeployment.Name, httpbinDeployment.Port)
		require.NoError(t, err)

		// Wait for the Gateway to get an address (Istio creates a Service/LB for it)
		addr, err := gatewayapihelper.GetGatewayAddress(t, gatewayName, defaultNamespace)
		require.NoError(t, err)

		httpClient := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("gateway-api-test"),
		)

		httpassert.AssertOKResponse(t, httpClient, fmt.Sprintf("http://%s/status/200", addr))
	})
}




