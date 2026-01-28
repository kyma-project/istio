package observability

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	logsassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/logs"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	observabilityhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/observability"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

const (
	defaultNamespace = "default"
)

func TestObservability(t *testing.T) {
	t.Run("Logs from stdout-json envoyFileAccessLog provider are in correct format", func(t *testing.T) {
		// given
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = observabilityhelper.SetupLogs(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		httpbinInfo, err := httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"httpbin",
			defaultNamespace,
			httpbinInfo.Host,
			httpbinInfo.Host,
			gatewayhelper.GatewayReference,
		)
		require.NoError(t, err)

		ip, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		// when
		httpClient := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("observability-test"),
			httphelper.WithHost(httpbinInfo.Host),
		)
		url := fmt.Sprintf("http://%s/headers", ip)
		httpassert.AssertOKResponse(t, httpClient, url)

		// then
		logsassert.AssertIstioProxyLogsContain(
			t,
			c,
			httpbinInfo.WorkloadSelector,
			observabilityhelper.EnvoyAccessLogFields(),
		)
	})

	t.Run("Istio calls OpenTelemetry API on default service configured in kyma-traces extension provider", func(t *testing.T) {
		// given
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		otelCollectorInfo, err := observabilityhelper.SetupTraces(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		httpbinInfo, err := httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"httpbin",
			defaultNamespace,
			httpbinInfo.Host,
			httpbinInfo.Host,
			gatewayhelper.GatewayReference,
		)
		require.NoError(t, err)

		ip, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		// when
		httpClient := httphelper.NewHTTPClient(
			t,
			httphelper.WithPrefix("observability-test"),
			httphelper.WithHost(httpbinInfo.Host),
		)
		httpassert.AssertOKResponse(t, httpClient, fmt.Sprintf("http://%s/headers", ip))

		// then
		logsassert.AssertContainerLogContainsWithRetry(
			t,
			c,
			otelCollectorInfo.WorkloadSelector,
			otelCollectorInfo.Namespace,
			otelCollectorInfo.ContainerName,
			"POST /opentelemetry.proto.collector.trace.v1.TraceService/Export",
		)
	})
}
