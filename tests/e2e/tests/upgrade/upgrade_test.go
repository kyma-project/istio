package upgrade

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

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
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/sidecar"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

func TestUpgrade(t *testing.T) {

	t.Run("Upgrade module version", func(t *testing.T) {
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

		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtual_service.CreateVirtualService(
			t,
			"httpbin-vs",
			"default",
			httpbinNativeSidecar.Host,
			[]string{httpbinNativeSidecar.Host},
			[]string{"kyma-system/kyma-gateway"},
		)
		require.NoError(t, err)

		err = virtual_service.CreateVirtualService(
			t,
			"httpbin-vs-regular-sidecar",
			"default",
			httpbinRegularSidecar.Host,
			[]string{httpbinRegularSidecar.Host},
			[]string{"kyma-system/kyma-gateway"},
		)
		require.NoError(t, err)

		httpbinPodList := &v1.PodList{}
		err = c.List(t.Context(), httpbinPodList, resources.WithLabelSelector("app=httpbin"))
		require.NoError(t, err)

		httpbinRegularPodList := &v1.PodList{}
		err = c.List(t.Context(), httpbinRegularPodList, resources.WithLabelSelector("app=httpbin-regular-sidecar"))
		require.NoError(t, err)

		for _, pod := range httpbinPodList.Items {
			err = sidecar.VerifyIfPodHasIstioSidecar(&pod)
			require.NoError(t, err)
		}

		for _, pod := range httpbinRegularPodList.Items {
			err = sidecar.VerifyIfPodHasIstioSidecar(&pod)
			require.NoError(t, err)
		}

		lbIp, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		t.Logf("LoadBalancer IP: %s", lbIp)

		httpClient := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("upgrade-test"),
			httphelper.WithHost(httpbinNativeSidecar.Host),
		)

		httpassert.AssertOKResponse(t, httpClient, fmt.Sprintf("http://%s/headers", lbIp),
			httpassert.WithTimeout(60*time.Second),
		)

		httpClient = httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("upgrade-test"),
			httphelper.WithHost(httpbinRegularSidecar.Host),
		)

		httpassert.AssertOKResponse(t, httpClient, fmt.Sprintf("http://%s/headers", lbIp),
			httpassert.WithTimeout(60*time.Second),
		)

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

		t.Log("Stopping zero downtime tests and checking for errors")
		_, err = zeroDowntimeRunner.FinishZeroDowntimeTests(t.Context())
		require.NoError(t, err)
		t.Log("Zero downtime tests completed successfully - no downtime detected during upgrade")

	})

}
