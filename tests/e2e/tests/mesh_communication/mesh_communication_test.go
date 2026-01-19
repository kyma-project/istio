package mesh_communication

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"

	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"

	extauth "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/nginx"
)

func TestMeshCommunication(t *testing.T) {
	t.Run("Access between applications in different namespaces", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrastructure.CreateNamespace(
			t,
			"target",
			infrastructure.WithSidecarInjectionEnabled(),
		)
		require.NoError(t, err)

		httpbin, err := httpbin.NewBuilder().WithNamespace("target").DeployWithCleanup(t)
		require.NoError(t, err)

		err = infrastructure.CreateNamespace(
			t,
			"source",
			infrastructure.WithSidecarInjectionEnabled(),
		)
		require.NoError(t, err)

		sourceWorkloadUrl, err := nginx.CreateForwardRequestNginx(t, "nginx-mesh-communication", "source", fmt.Sprintf("%s:%d", httpbin.Host, httpbin.Port))
		require.NoError(t, err)

		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtual_service.CreateVirtualService(t, "nginx-mesh-communication", "source", sourceWorkloadUrl, []string{sourceWorkloadUrl}, []string{"kyma-system/kyma-gateway"})
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
			"target",
			infrastructure.WithSidecarInjectionEnabled(),
		)
		require.NoError(t, err)

		httpbin, err := httpbin.NewBuilder().WithNamespace("target").DeployWithCleanup(t)
		require.NoError(t, err)

		// source should not be istio injected
		err = infrastructure.CreateNamespace(
			t,
			"source",
		)
		require.NoError(t, err)

		sourceWorkloadUrl, err := nginx.CreateForwardRequestNginx(t, "nginx-mesh-communication", "source", fmt.Sprintf("%s:%d", httpbin.Host, httpbin.Port))
		require.NoError(t, err)

		err = extauth.CreateHTTPGateway(t)
		require.NoError(t, err)

		ip, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())

		err = virtual_service.CreateVirtualService(t, "nginx-mesh-communication", "source", sourceWorkloadUrl, []string{sourceWorkloadUrl}, []string{"kyma-system/kyma-gateway"})
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
			"sidecar-disabled",
			infrastructure.WithSidecarInjectionDisabled(),
		)
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().WithNamespace("sidecar-disabled").DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinPodList := &v1.PodList{}
		err = c.List(t.Context(), httpbinPodList, resources.WithLabelSelector("app=httpbin"))
		require.NoError(t, err)

		for _, pod := range httpbinPodList.Items {
			for _, container := range pod.Spec.InitContainers {
				require.NotEqual(t, "istio-proxy", container.Name, "Found istio-proxy sidecar in pod %s", pod.Name)
			}
		}
	})

	t.Run("Namespace with istio-injection=enabled label does contain pods with istio sidecar", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		err = infrastructure.CreateNamespace(
			t,
			"sidecar-enabled",
			infrastructure.WithSidecarInjectionEnabled(),
		)
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().WithNamespace("sidecar-enabled").DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinPodList := &v1.PodList{}
		err = c.List(t.Context(), httpbinPodList, resources.WithLabelSelector("app=httpbin"))
		require.NoError(t, err)

		for _, pod := range httpbinPodList.Items {
			contain := false
			for _, container := range pod.Spec.InitContainers {
				if container.Name == "istio-proxy" {
					contain = true
					continue
				}
			}
			require.True(t, contain)
		}
	})

	t.Run("Kube-system namespace does not contain pods with sidecar", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().WithNamespace("kube-system").DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinPodList := &v1.PodList{}
		err = c.List(t.Context(), httpbinPodList, resources.WithLabelSelector("app=httpbin"))
		require.NoError(t, err)

		for _, pod := range httpbinPodList.Items {
			if pod.Namespace == "kube-system" {
				for _, container := range pod.Spec.InitContainers {
					require.NotEqual(t, "istio-proxy", container.Name)
				}
			}
		}
	})
}
