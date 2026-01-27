package observability

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"testing"
	"time"

	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/log"

	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"

	extauth "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/telemetry"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"

	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
)


func TestObservability(t *testing.T) {
	t.Run("Logs from stdout-json envoyFileAccessLog provider are in correct format", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = telemetry.EnableLogs(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		httpbin, err := httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtual_service.CreateVirtualService(
			t,
			"httpbin",
			"default",
			httpbin.Host,
			[]string{httpbin.Host},
			[]string{"kyma-system/kyma-gateway"},
		)
		require.NoError(t, err)

		ip, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())

		httpClient := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("observability-test"),
			httphelper.WithHost(httpbin.Host),
		)
		url := fmt.Sprintf("http://%s/headers", ip)
		httpassert.AssertOKResponse(t, httpClient, url)

		httpbinPods := v1.PodList{}
		err = c.List(t.Context(), &httpbinPods, resources.WithLabelSelector(httpbin.WorkloadSelector))
		require.NoError(t, err)

		requiredLogEntries := []string{
			"start_time",
			"method",
			"path",
			"protocol",
			"response_code",
			"response_flags",
			"response_code_details",
			"connection_termination_details",
			"upstream_transport_failure_reason",
			"bytes_received",
			"bytes_sent",
			"duration",
			"upstream_service_time",
			"x_forwarded_for",
			"user_agent",
			"request_id",
			"authority",
			"upstream_host",
			"upstream_cluster",
			"upstream_local_address",
			"downstream_local_address",
			"downstream_remote_address",
			"requested_server_name",
			"route_name",
			"traceparent",
			"tracestate",
		}

		for _, pod := range httpbinPods.Items {
			logs, err := log.GetLogsFromIstioProxy(t, pod.Name, pod.Namespace)
			require.NoError(t, err)

			for _, entry := range requiredLogEntries {
				require.Containsf(t, string(logs), entry, "Log entry %s not found in logs from pod %s", entry, pod.Name)
			}
		}
	})

	t.Run("Istio calls OpenTelemetry API on default service configured in kyma-traces extension provider", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = telemetry.EnableTraces(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		httpbin, err := httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		// create gateway
		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		// when
		err = virtual_service.CreateVirtualService(
			t,
			"httpbin",
			"default",
			httpbin.Host,
			[]string{httpbin.Host},
			[]string{"kyma-system/kyma-gateway"},
		)
		require.NoError(t, err)

		err = telemetry.CreateOtelMockCollector(t)
		require.NoError(t, err)

		ip, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())

		httpClient := httphelper.NewHTTPClient(
			t,
			httphelper.WithPrefix("observability-test"),
			httphelper.WithHost(httpbin.Host),
			)
		url := fmt.Sprintf("http://%s/headers", ip)
		httpassert.AssertOKResponse(t, httpClient, url)

		otelCollectorMockPods := v1.PodList{}
		err = c.List(t.Context(), &otelCollectorMockPods, resources.WithLabelSelector("app=otel-collector-mock"))
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			t.Logf("Waiting for logs to appear in the otel-collector-mock")

			for _, pod := range otelCollectorMockPods.Items {
				logs, err := log.GetLogsFromPodContainer(t, pod.Name, pod.Namespace, "otel-collector-mock")
				if err != nil {
					t.Logf("Failed to get logs from pod container: %v", err)
					return false, err
				}
				if !strings.Contains(string(logs), "POST /opentelemetry.proto.collector.trace.v1.TraceService/Export") {
					t.Logf("Log entry %s not found in logs from pod %s", string(logs), pod.Name)
					return false, nil
				}
			}
			return true, nil
		}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))
	})
}
