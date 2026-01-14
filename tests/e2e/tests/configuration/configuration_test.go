package configuration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	apisecurityv1 "istio.io/api/security/v1"
	apiv1beta1 "istio.io/api/type/v1beta1"
	securityv1 "istio.io/client-go/pkg/apis/security/v1"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/extauth"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

func TestConfiguration(t *testing.T) {
	t.Run("Updating proxy resource configuration", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)
		istioCR, err := modulehelpers.NewIstioCRBuilder().
			WithProxyResources("30m", "190Mi", "700m", "700Mi").
			ApplyAndCleanup(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		_, err = httpbin.NewBuilder().WithName("httpbin-regular-sidecar").WithRegularSidecar().DeployWithCleanup(t)
		require.NoError(t, err)

		assertProxyResourcesForDeployment(t, c, "httpbin", "default", "30m", "190Mi", "700m", "700Mi")

		assertProxyResourcesForDeployment(t, c, "httpbin-regular-sidecar", "default", "30m", "190Mi", "700m", "700Mi")

		err = modulehelpers.NewIstioCRBuilder().
			WithName(istioCR.Name).
			WithNamespace(istioCR.Namespace).
			WithProxyResources("80m", "230Mi", "900m", "900Mi").
			Update(t)
		require.NoError(t, err)

		assertProxyResourcesForDeployment(t, c, "httpbin", "default", "80m", "230Mi", "900m", "900Mi")
		assertProxyResourcesForDeployment(t, c, "httpbin-regular-sidecar", "default", "80m", "230Mi", "900m", "900Mi")
	})

	t.Run("Ingress Gateway adds correct X-Envoy-External-Address header after updating numTrustedProxies", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		istioCR, err := modulehelpers.NewIstioCRBuilder().
			WithNumTrustedProxies(1).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		httpbinInfo, err := httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"test-vs",
			"default",
			httpbinInfo.Host,
			[]string{httpbinInfo.Host},
			[]string{"kyma-system/kyma-gateway"},
		)
		require.NoError(t, err)

		gatewayAddress, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		assertEnvoyExternalAddress(t, gatewayAddress, httpbinInfo.Host, "10.2.1.1,10.0.0.1", "10.0.0.1")

		err = modulehelpers.NewIstioCRBuilder().
			WithName(istioCR.Name).
			WithNamespace(istioCR.Namespace).
			WithNumTrustedProxies(2).
			Update(t)
		require.NoError(t, err)

		assertEnvoyExternalAddress(t, gatewayAddress, httpbinInfo.Host, "10.2.1.1,10.0.0.1", "10.2.1.1")
	})

	t.Run("Egress Gateway has correct configuration", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		enabled := true
		_, err = modulehelpers.NewIstioCRBuilder().
			WithEgressGateway(&v1alpha2.EgressGateway{
				Enabled: &enabled,
			}).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		egressDeployment, err := infrahelpers.GetEgressGatewayDeployment(t)
		err = wait.For(conditions.New(c).DeploymentConditionMatch(egressDeployment, v1.DeploymentAvailable, corev1.ConditionTrue), wait.WithContext(t.Context()))
		require.NoError(t, err)
	})

	t.Run("External authorizer", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		extAuth, err := extauth.NewBuilder().WithName("ext-authz").WithNamespace("ext-auth").DeployWithCleanup(t)
		require.NoError(t, err)

		extAuth2, err := extauth.NewBuilder().WithName("ext-authz2").WithNamespace("ext-auth").DeployWithCleanup(t)
		require.NoError(t, err)

		authorizer1 := &v1alpha2.Authorizer{
			Name:    "ext-authz",
			Port:    uint32(extAuth.HttpPort),
			Service: extAuth.Host,
			Headers: &v1alpha2.Headers{
				InCheck: &v1alpha2.InCheck{
					Include: []string{"X-Ext-Authz"},
					Add: map[string]string{
						"X-Add-In-Check": "value",
					},
				},
			},
		}
		authorizer2 := &v1alpha2.Authorizer{
			Name:    "ext-authz2",
			Port:    uint32(extAuth2.HttpPort),
			Service: extAuth2.Host,
			Headers: &v1alpha2.Headers{
				InCheck: &v1alpha2.InCheck{
					Include: []string{"X-Ext-Authz"},
					Add: map[string]string{
						"X-Add-In-Check": "value",
					},
				},
			},
		}
		_, err = modulehelpers.NewIstioCRBuilder().
			WithAuthorizer(authorizer1).
			WithAuthorizer(authorizer2).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		httpbinInfo, err := httpbin.NewBuilder().WithName("httpbin-ext-auth").DeployWithCleanup(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"httpbin-ext-auth",
			"default",
			httpbinInfo.Host,
			[]string{httpbinInfo.Host},
			[]string{"kyma-system/kyma-gateway"},
		)
		require.NoError(t, err)

		err = createAuthorizationPolicyExtAuthz(t, "ext-authz", "default", "httpbin-ext-auth", "ext-authz", "/headers")
		require.NoError(t, err)

		httpbin2Info, err := httpbin.NewBuilder().WithName("httpbin-ext-auth2").DeployWithCleanup(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"httpbin-ext-auth2",
			"default",
			httpbin2Info.Host,
			[]string{httpbin2Info.Host},
			[]string{"kyma-system/kyma-gateway"},
		)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = createAuthorizationPolicyExtAuthz(t, "ext-authz2", "default", "httpbin-ext-auth2", "ext-authz2", "/headers")
		require.NoError(t, err)

		gatewayAddress, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			hc := httphelper.NewHTTPClient(t, httphelper.WithHost(httpbinInfo.Host))

			url := fmt.Sprintf("http://%s/", gatewayAddress)
			resp, err := hc.Get(url)
			if err != nil {
				return false, err
			}

			if resp.StatusCode != http.StatusOK {
				t.Logf("Expected status code %d got status code %d", http.StatusOK, resp.StatusCode)
				return false, nil
			}

			return true, nil
		}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			hc := httphelper.NewHTTPClient(t,
				httphelper.WithHost(httpbinInfo.Host),
				httphelper.WithHeaders(map[string]string{"x-ext-authz": "allow"}),
			)

			url := fmt.Sprintf("http://%s/headers", gatewayAddress)
			resp, err := hc.Get(url)
			if err != nil {
				return false, err
			}

			if resp.StatusCode != http.StatusOK {
				t.Logf("Expected status code %d got status code %d", http.StatusOK, resp.StatusCode)
				return false, nil
			}

			return true, nil
		}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			hc := httphelper.NewHTTPClient(t,
				httphelper.WithHost(httpbinInfo.Host),
				httphelper.WithHeaders(map[string]string{"x-ext-authz": "deny"}),
			)

			url := fmt.Sprintf("http://%s/headers", gatewayAddress)
			resp, err := hc.Get(url)
			if err != nil {
				return false, err
			}

			if resp.StatusCode != http.StatusForbidden {
				t.Logf("Expected status code %d got status code %d", http.StatusOK, resp.StatusCode)
				return false, nil
			}

			return true, nil
		}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))
		require.NoError(t, err)

		// now request to httpbin-ext-auth2
		err = wait.For(func(ctx context.Context) (done bool, err error) {
			hc := httphelper.NewHTTPClient(t, httphelper.WithHost(httpbin2Info.Host))

			url := fmt.Sprintf("http://%s/", gatewayAddress)
			resp, err := hc.Get(url)
			if err != nil {
				return false, err
			}

			if resp.StatusCode != http.StatusOK {
				t.Logf("Expected status code %d got status code %d", http.StatusOK, resp.StatusCode)
				return false, nil
			}

			return true, nil
		}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			hc := httphelper.NewHTTPClient(t,
				httphelper.WithHost(httpbin2Info.Host),
				httphelper.WithHeaders(map[string]string{"x-ext-authz": "allow"}),
			)

			url := fmt.Sprintf("http://%s/headers", gatewayAddress)
			resp, err := hc.Get(url)
			if err != nil {
				return false, err
			}

			if resp.StatusCode != http.StatusOK {
				t.Logf("Expected status code %d got status code %d", http.StatusOK, resp.StatusCode)
				return false, nil
			}

			return true, nil
		}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))
		require.NoError(t, err)

		err = wait.For(func(ctx context.Context) (done bool, err error) {
			hc := httphelper.NewHTTPClient(t,
				httphelper.WithHost(httpbin2Info.Host),
				httphelper.WithHeaders(map[string]string{"x-ext-authz": "deny"}),
			)

			url := fmt.Sprintf("http://%s/headers", gatewayAddress)
			resp, err := hc.Get(url)
			if err != nil {
				return false, err
			}

			if resp.StatusCode != http.StatusForbidden {
				t.Logf("Expected status code %d got status code %d", http.StatusOK, resp.StatusCode)
				return false, nil
			}

			return true, nil
		}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))
		require.NoError(t, err)
	})
}

func assertProxyResourcesForDeployment(t *testing.T, c *resources.Resources, deploymentName, _ string, cpuRequest, memRequest, cpuLimit, memLimit string) {
	t.Helper()

	// Wait for the deployment to be restarted with new resource configurations
	err := wait.For(func(ctx context.Context) (done bool, err error) {
		podList := &corev1.PodList{}
		err = c.List(ctx, podList, resources.WithLabelSelector(fmt.Sprintf("app=%s", deploymentName)))
		if err != nil {
			return false, err
		}

		if len(podList.Items) == 0 {
			return false, fmt.Errorf("no pods found for deployment %s", deploymentName)
		}

		for _, pod := range podList.Items {
			// Skip pods that are not ready
			if pod.Status.Phase != corev1.PodRunning {
				return false, nil
			}

			// Check init containers for istio-proxy
			for _, container := range pod.Spec.InitContainers {
				if container.Name == "istio-proxy" {
					if !checkResourceValues(container.Resources, cpuRequest, memRequest, cpuLimit, memLimit) {
						return false, nil
					}
					return true, nil
				}
			}

			// Check regular containers for istio-proxy
			for _, container := range pod.Spec.Containers {
				if container.Name == "istio-proxy" {
					if !checkResourceValues(container.Resources, cpuRequest, memRequest, cpuLimit, memLimit) {
						return false, nil
					}
					return true, nil
				}
			}
		}

		return false, fmt.Errorf("istio-proxy container not found in pods for deployment %s", deploymentName)
	}, wait.WithTimeout(1*time.Minute), wait.WithInterval(5*time.Second), wait.WithContext(t.Context()))

	require.NoError(t, err, "Failed to verify proxy resources for deployment %s", deploymentName)
}

func checkResourceValues(resources corev1.ResourceRequirements, cpuRequest, memRequest, cpuLimit, memLimit string) bool {
	expectedCPURequest := resource.MustParse(cpuRequest)
	expectedMemRequest := resource.MustParse(memRequest)
	expectedCPULimit := resource.MustParse(cpuLimit)
	expectedMemLimit := resource.MustParse(memLimit)

	actualCPURequest := resources.Requests[corev1.ResourceCPU]
	actualMemRequest := resources.Requests[corev1.ResourceMemory]
	actualCPULimit := resources.Limits[corev1.ResourceCPU]
	actualMemLimit := resources.Limits[corev1.ResourceMemory]

	return actualCPURequest.Equal(expectedCPURequest) &&
		actualMemRequest.Equal(expectedMemRequest) &&
		actualCPULimit.Equal(expectedCPULimit) &&
		actualMemLimit.Equal(expectedMemLimit)
}

func assertEnvoyExternalAddress(t *testing.T, gatewayAddress string, hostHeader string, xForwardedFor, expectedExternalAddress string) {
	t.Helper()

	httpClient := httphelper.NewHTTPClient(t, httphelper.WithHost(hostHeader))

	err := wait.For(func(ctx context.Context) (done bool, err error) {
		url := fmt.Sprintf("http://%s/headers", gatewayAddress)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return false, err
		}

		req.Header.Set("X-Forwarded-For", xForwardedFor)

		resp, err := httpClient.Do(req)
		if err != nil {
			return false, err
		}
		defer func() {
			_ = resp.Body.Close()
		}()

		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		// Parse JSON response body
		var bodyResponse struct {
			Headers map[string][]string `json:"headers"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&bodyResponse); err != nil {
			return false, fmt.Errorf("failed to decode response body: %w", err)
		}

		// Get X-Envoy-External-Address from the headers in the body
		externalAddressValues, ok := bodyResponse.Headers["X-Envoy-External-Address"]
		if !ok || len(externalAddressValues) == 0 {
			return false, fmt.Errorf("X-Envoy-External-Address not found in response body")
		}

		actualExternalAddress := externalAddressValues[0]
		if actualExternalAddress != expectedExternalAddress {
			return false, fmt.Errorf("X-Envoy-External-Address mismatch: expected %s, got %s", expectedExternalAddress, actualExternalAddress)
		}

		return true, nil
	}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))

	require.NoError(t, err)
}

func createAuthorizationPolicyExtAuthz(t *testing.T, name, namespace, selector, provider, operation string) error {
	t.Helper()

	c, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	ap := &securityv1.AuthorizationPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apisecurityv1.AuthorizationPolicy{
			Selector: &apiv1beta1.WorkloadSelector{
				MatchLabels: map[string]string{"app": selector},
			},
			Action: apisecurityv1.AuthorizationPolicy_CUSTOM,
			ActionDetail: &apisecurityv1.AuthorizationPolicy_Provider{
				Provider: &apisecurityv1.AuthorizationPolicy_ExtensionProvider{
					Name: provider,
				},
			},
			Rules: []*apisecurityv1.Rule{
				{
					To: []*apisecurityv1.Rule_To{
						{
							Operation: &apisecurityv1.Operation{
								Paths: []string{operation},
							},
						},
					},
				},
			},
		},
	}

	setup.DeclareCleanup(t, func() {
		t.Logf("Cleaning up authorization policy %s in namespace %s", ap.GetName(), ap.GetNamespace())
		err := c.Delete(setup.GetCleanupContext(), ap)
		if err != nil {
			t.Logf("Failed to delete resource %s: %v", ap.GetName(), err)
			return
		}
	})

	err = c.Create(t.Context(), ap)
	if err != nil {
		return err
	}

	return nil
}
