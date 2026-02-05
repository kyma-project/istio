package smoke_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	extauth "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

const (
	defaultNamespace = "default"
)

func TestSmoke(t *testing.T) {
	t.Run("Httpbin is accessible through Istio Gateway", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(defaultNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		istioassert.AssertIstioProxyPresent(t, c, httpbinDeployment.WorkloadSelector)

		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtual_service.CreateVirtualService(t, "httpbin", defaultNamespace, httpbinDeployment.Host, httpbinDeployment.Host, extauth.GatewayReference)
		require.NoError(t, err)

		ip, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		httpClient := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("smoke-test"),
			httphelper.WithHost(httpbinDeployment.Host),
		)

		httpassert.AssertOKResponse(t, httpClient, fmt.Sprintf("http://%s/status/200", ip))
	})
}
