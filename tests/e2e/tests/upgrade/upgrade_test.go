package upgrade

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"

	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/zero_downtime"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/crds"
	extauth "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

func TestUpgrade(t *testing.T) {

	t.Run("Upgrade module version", func(t *testing.T) {
		//given
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		istioCR, err := modules.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = crds.AssertIstioCRDsPresent(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = c.Get(t.Context(), istioCR.Name, istioCR.Namespace, istioCR)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		httpbinNativeSidecar, err := httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinRegularSidecar, err := httpbin.NewBuilder().WithName("httpbin-regular-sidecar").WithRegularSidecar().DeployWithCleanup(t)
		require.NoError(t, err)

		istioassert.AssertIstioProxyPresent(t, c, httpbinNativeSidecar.WorkloadSelector)
		istioassert.AssertIstioProxyPresent(t, c, httpbinRegularSidecar.WorkloadSelector)

		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtual_service.CreateVirtualService(
			t,
			"httpbin-vs",
			"default",
			httpbinNativeSidecar.Host,
			httpbinNativeSidecar.Host,
			extauth.GatewayReference,
		)
		require.NoError(t, err)

		err = virtual_service.CreateVirtualService(
			t,
			"httpbin-vs-regular-sidecar",
			"default",
			httpbinRegularSidecar.Host,
			httpbinRegularSidecar.Host,
			extauth.GatewayReference,
		)
		require.NoError(t, err)

		lbIp, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		t.Logf("LoadBalancer IP: %s", lbIp)

		httpClient := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("upgrade-test"),
			httphelper.WithHost(httpbinNativeSidecar.Host),
		)

		httpassert.AssertOKResponse(t, httpClient, fmt.Sprintf("http://%s/headers", lbIp))

		httpClient = httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("upgrade-test"),
			httphelper.WithHost(httpbinRegularSidecar.Host),
		)

		httpassert.AssertOKResponse(t, httpClient, fmt.Sprintf("http://%s/headers", lbIp))

		// when
		t.Log("Starting zero downtime tests")
		zeroDowntimeRunner := &zero_downtime.ZeroDowntimeTestRunner{}

		_, err = zeroDowntimeRunner.StartZeroDowntimeTest(t.Context(), c.GetControllerRuntimeClient(), httpbinNativeSidecar.Host, "/headers")
		require.NoError(t, err)

		_, err = zeroDowntimeRunner.StartZeroDowntimeTest(t.Context(), c.GetControllerRuntimeClient(), httpbinRegularSidecar.Host, "/headers")
		require.NoError(t, err)

		t.Log("Starting Istio module upgrade")
		err = modules.UpgradeIstioModule(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)
		t.Log("Istio module upgrade completed successfully")

		//then
		t.Log("Stopping zero downtime tests and checking for errors")
		_, err = zeroDowntimeRunner.FinishZeroDowntimeTests(t.Context())
		require.NoError(t, err)
		t.Log("Zero downtime tests completed successfully - no downtime detected during upgrade")

		istioassert.AssertIstiodReady(t, c)
		istioassert.AssertIngressGatewayReady(t, c)
		istioassert.AssertCNINodeReady(t, c)

		t.Log("======Verifying component versions after upgrade======")
		istioassert.AssertIstiodContainerVersion(t, c)
		istioassert.AssertIngressGatewayContainerVersion(t, c)
		istioassert.AssertCNINodeContainerVersion(t, c)
		istioassert.AssertIstioProxyVersion(t, c, httpbinNativeSidecar.WorkloadSelector)
		istioassert.AssertIstioProxyVersion(t, c, httpbinRegularSidecar.WorkloadSelector)
		t.Log("======All components have the required version after upgrade======")
	})

}
