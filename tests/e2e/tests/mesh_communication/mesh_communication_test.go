package mesh_communication

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"

	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"

	extauth "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/nginx"
)

const (
	targetNamespace          = "target"
	sourceNamespace          = "source"
	sidecarDisabledNamespace = "sidecar-disabled"
	sidecarEnabledNamespace  = "sidecar-enabled"
	kubeSystemNamespace      = "kube-system"
)

func TestMeshCommunication(t *testing.T) {
	t.Run("Access between applications in different namespaces", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrastructure.CreateNamespace(
			t,
			targetNamespace,
			infrastructure.WithSidecarInjectionEnabled(),
		)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(targetNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		err = infrastructure.CreateNamespace(
			t,
			sourceNamespace,
			infrastructure.WithSidecarInjectionEnabled(),
		)
		require.NoError(t, err)

		sourceWorkloadUrl, err := nginx.CreateForwardRequestNginx(t, "nginx-mesh-communication", sourceNamespace, fmt.Sprintf("%s:%d", httpbinDeployment.Host, httpbinDeployment.Port))
		require.NoError(t, err)

		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtual_service.CreateVirtualService(t, "nginx-mesh-communication", sourceNamespace, sourceWorkloadUrl, []string{sourceWorkloadUrl}, []string{"kyma-system/kyma-gateway"})
		require.NoError(t, err)

		ip, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		httpClient := httphelper.NewHTTPClient(
			t,
			httphelper.WithPrefix("mesh-communication-test"),
			httphelper.WithHost(sourceWorkloadUrl),
		)

		httpassert.AssertOKResponse(t, httpClient, fmt.Sprintf("http://%s/headers", ip),
			httpassert.WithExpectedBodyContains("httpbin.target.svc.cluster.local"),
		)

	})

	t.Run("Access between applications from injection disabled namespace to injection enabled namespace is restricted", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrastructure.CreateNamespace(
			t,
			targetNamespace,
			infrastructure.WithSidecarInjectionEnabled(),
		)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(targetNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		// source should not be istio injected
		err = infrastructure.CreateNamespace(
			t,
			sourceNamespace,
		)
		require.NoError(t, err)

		sourceWorkloadUrl, err := nginx.CreateForwardRequestNginx(t, "nginx-mesh-communication", sourceNamespace, fmt.Sprintf("%s:%d", httpbinDeployment.Host, httpbinDeployment.Port))
		require.NoError(t, err)

		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		ip, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())

		err = virtual_service.CreateVirtualService(t, "nginx-mesh-communication", sourceNamespace, sourceWorkloadUrl, []string{sourceWorkloadUrl}, []string{"kyma-system/kyma-gateway"})
		require.NoError(t, err)

		httpClient := httphelper.NewHTTPClient(
			t,
			httphelper.WithPrefix("mesh-communication-test"),
			httphelper.WithHost(sourceWorkloadUrl),
		)

		httpassert.AssertResponse(t, httpClient, fmt.Sprintf("http://%s/", ip),
			httpassert.WithExpectedStatusCode(502),
		)
	})

	t.Run("Namespace with istio-injection=disabled label does not contain pods with istio sidecar", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrastructure.CreateNamespace(
			t,
			sidecarDisabledNamespace,
			infrastructure.WithSidecarInjectionDisabled(),
		)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(sidecarDisabledNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		istioassert.AssertIstioProxyAbsent(t, c, httpbinDeployment.WorkloadSelector)
	})

	t.Run("Namespace with istio-injection=enabled label contain pods with istio sidecar", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrastructure.CreateNamespace(
			t,
			sidecarEnabledNamespace,
			infrastructure.WithSidecarInjectionEnabled(),
		)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(sidecarEnabledNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		istioassert.AssertIstioProxyPresent(t, c, httpbinDeployment.WorkloadSelector)
	})

	t.Run("Kube-system namespace does not contain pods with sidecar", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		httpbinDeployment, err := httpbin.NewBuilder().WithNamespace(kubeSystemNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		istioassert.AssertIstioProxyAbsent(t, c, httpbinDeployment.WorkloadSelector)
	})
}
