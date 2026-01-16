package mesh_communication

import (
	"context"
	_ "embed"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"

	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"

	extauth "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/nginx"
)

//go:embed virtual_service_nginx.yaml
var VirtualServiceSourceWorkload string

func TestMeshCommunication(t *testing.T) {
	t.Run("Access between applications in different namespaces", func(t *testing.T) {
		_, err := client.ResourcesClient(t)
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

		createdVs, err := infrastructure.CreateResourceWithTemplateValues(
			t,
			VirtualServiceSourceWorkload,
			map[string]any{
				"Name":            "nginx-mesh-communication",
				"GatewayName":     "kyma-system/kyma-gateway",
				"HostName":        "nginx-mesh-communication.local.kyma.dev",
				"DestinationHost": sourceWorkloadUrl,
				"DestinationPort": 80,
			},
			decoder.MutateNamespace("source"),
		)
		require.NoError(t, err)
		require.NotEmpty(t, createdVs)

		//_, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		//require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			t.Logf("Waiting for endpoint to return 200 OK")
			httpClient := httphelper.NewHTTPClient(t, httphelper.WithPrefix("mesh-communication-test"))

			resp, err := httpClient.Get("http://nginx-mesh-communication.local.kyma.dev/headers")
			if err != nil {
				return false, err
			}
			if resp.StatusCode != 200 {
				t.Logf("Unexpected status code: %d", resp.StatusCode)
				return false, nil
			}

			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Logf("Failed to read response body: %v", err)
				return false, err
			}
			contains := strings.Contains(string(respBody), "httpbin.target.svc.cluster.local")
			if !contains {
				t.Logf("Endpoint not found in response: %s", string(respBody))
			} else {
				t.Logf("Endpoint found in response: %s", string(respBody))
			}

			return true, nil
		})

	})

	t.Run("Access between applications from injection disabled namespace to injection enabled namespace is restricted", func(t *testing.T) {
		_, err := modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)
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

		createdVs, err := infrastructure.CreateResourceWithTemplateValues(
			t,
			VirtualServiceSourceWorkload,
			map[string]any{
				"Name":            "nginx-mesh-communication",
				"GatewayName":     "kyma-system/kyma-gateway",
				"HostName":        "nginx-mesh-communication.local.kyma.dev",
				"DestinationHost": sourceWorkloadUrl,
				"DestinationPort": 80,
			},
			decoder.MutateNamespace("source"),
		)
		require.NoError(t, err)
		require.NotEmpty(t, createdVs)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			t.Logf("Waiting for endpoint to return 200 OK")
			httpClient := httphelper.NewHTTPClient(t, httphelper.WithPrefix("mesh-communication-test"))

			resp, err := httpClient.Get("http://nginx-mesh-communication.local.kyma.dev/headers")
			if err != nil {
				return false, err
			}
			if resp.StatusCode != 502 {
				t.Logf("Unexpected status code: %d", resp.StatusCode)
				return false, nil
			}

			return true, nil
		})
	})

	t.Run("Namespace with istio-injection=disabled label does not contain pods with istio sidecar", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)

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
		_, err = modulehelpers.NewIstioCRBuilder().ApplyAndCleanup(t)

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
