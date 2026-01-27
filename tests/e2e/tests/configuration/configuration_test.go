package configuration

import (
	"context"
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

	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/extauth"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

func TestConfiguration(t *testing.T) {
	t.Run("Updating proxy resource configuration", func(t *testing.T) {
		// given
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
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

		// when
		err = modulehelpers.NewIstioCRBuilder().
			WithName(istioCR.Name).
			WithNamespace(istioCR.Namespace).
			WithProxyResources("80m", "230Mi", "900m", "900Mi").
			Update(t)
		require.NoError(t, err)

		//then
		assertProxyResourcesForDeployment(t, c, "httpbin", "default", "80m", "230Mi", "900m", "900Mi")
		assertProxyResourcesForDeployment(t, c, "httpbin-regular-sidecar", "default", "80m", "230Mi", "900m", "900Mi")
	})

	t.Run("Ingress Gateway adds correct X-Envoy-External-Address header after updating numTrustedProxies", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
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
			httpbinInfo.Host,
			"kyma-system/kyma-gateway",
		)
		require.NoError(t, err)

		gatewayAddress, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		hc := httphelper.NewHTTPClient(t,
			httphelper.WithHost(httpbinInfo.Host),
			httphelper.WithHeaders(map[string]string{"X-Forwarded-For": "10.2.1.1,10.0.0.1"}),
		)
		url := fmt.Sprintf("http://%s/headers", gatewayAddress)
		httpassert.AssertResponse(t, hc, url,
			httpassert.WithExpectedStatusCode(http.StatusOK),
			httpassert.WithExpectedBodyContains(`"X-Envoy-External-Address": [`, `"10.0.0.1"`),
		)

		// when
		err = modulehelpers.NewIstioCRBuilder().
			WithName(istioCR.Name).
			WithNamespace(istioCR.Namespace).
			WithNumTrustedProxies(2).
			Update(t)
		require.NoError(t, err)

		// then
		httpassert.AssertOKResponse(t, hc, url,
			httpassert.WithExpectedBodyContains(`"X-Envoy-External-Address": [`, `"10.2.1.1"`),
		)
	})

	t.Run("Egress Gateway has correct configuration", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		enabled := true
		_, err = modulehelpers.NewIstioCRBuilder().
			WithEgressGateway(&v1alpha2.EgressGateway{
				Enabled: &enabled,
			}).
			ApplyAndCleanup(t)
		require.NoError(t, err)

		egressDeployment, err := infrahelpers.GetEgressGatewayDeployment(t)
		//TODO: somepackage.AssertDeploymentReady(d *v1.Deployment) err
		err = wait.For(conditions.New(c).DeploymentConditionMatch(egressDeployment, v1.DeploymentAvailable, corev1.ConditionTrue), wait.WithContext(t.Context()))
		require.NoError(t, err)
	})

	t.Run("External authorizer", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, "default")
		require.NoError(t, err)

		extAuth, err := extauth.NewBuilder().WithName("ext-authz").WithNamespace("ext-auth").DeployWithCleanup(t)
		require.NoError(t, err)

		extAuth2, err := extauth.NewBuilder().WithName("ext-authz2").WithNamespace("ext-auth").DeployWithCleanup(t)
		require.NoError(t, err)

		//TODO: hide it
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
			httpbinInfo.Host,
			"kyma-system/kyma-gateway",
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
			httpbin2Info.Host,
			"kyma-system/kyma-gateway",
		)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = createAuthorizationPolicyExtAuthz(t, "ext-authz2", "default", "httpbin-ext-auth2", "ext-authz2", "/headers")
		require.NoError(t, err)

		gatewayAddress, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		hc := httphelper.NewHTTPClient(t, httphelper.WithHost(httpbinInfo.Host))
		url := fmt.Sprintf("http://%s/", gatewayAddress)
		httpassert.AssertOKResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithHost(httpbinInfo.Host),
			httphelper.WithHeaders(map[string]string{"x-ext-authz": "allow"}),
		)
		url = fmt.Sprintf("http://%s/headers", gatewayAddress)
		httpassert.AssertOKResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithHost(httpbinInfo.Host),
			httphelper.WithHeaders(map[string]string{"x-ext-authz": "deny"}),
		)
		httpassert.AssertForbiddenResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t, httphelper.WithHost(httpbin2Info.Host))
		url = fmt.Sprintf("http://%s/", gatewayAddress)
		httpassert.AssertOKResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithHost(httpbin2Info.Host),
			httphelper.WithHeaders(map[string]string{"x-ext-authz": "allow"}),
		)
		url = fmt.Sprintf("http://%s/headers", gatewayAddress)
		httpassert.AssertOKResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithHost(httpbin2Info.Host),
			httphelper.WithHeaders(map[string]string{"x-ext-authz": "deny"}),
		)
		httpassert.AssertForbiddenResponse(t, hc, url)
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
			if pod.Status.Phase != corev1.PodRunning {
				return false, nil
			}

			for _, container := range pod.Spec.InitContainers {
				if container.Name == "istio-proxy" {
					if !checkResourceValues(container.Resources, cpuRequest, memRequest, cpuLimit, memLimit) {
						return false, nil
					}
					return true, nil
				}
			}

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
