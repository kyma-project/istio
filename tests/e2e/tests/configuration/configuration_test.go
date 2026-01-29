package configuration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	httpbinassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/httpbin"
	istioassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/istio"

	authzpolicy "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/authorization_policy"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/extauth"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	httpassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/http"
	resourceassert "github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/resources"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/load_balancer"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/namespace"
	virtualservice "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/virtual_service"
)

const (
	defaultNamespace = "default"
	extAuthNamespace = "ext-auth"
	kymaGateway      = "kyma-system/kyma-gateway"
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

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		httpbinInfo, err := httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		httpbinRegularInfo, err := httpbin.NewBuilder().WithName("httpbin-regular-sidecar").WithRegularSidecar().DeployWithCleanup(t)
		require.NoError(t, err)

		resourceassert.AssertIstioProxyResourcesEventually(t, c, httpbinInfo.WorkloadSelector, "30m", "190Mi", "700m", "700Mi")
		resourceassert.AssertIstioProxyResourcesEventually(t, c, httpbinRegularInfo.WorkloadSelector, "30m", "190Mi", "700m", "700Mi")

		// when
		err = modulehelpers.NewIstioCRBuilder().
			WithName(istioCR.Name).
			WithNamespace(istioCR.Namespace).
			WithProxyResources("80m", "230Mi", "900m", "900Mi").
			Update(t)
		require.NoError(t, err)

		//then
		resourceassert.AssertIstioProxyResourcesEventually(t, c, httpbinInfo.WorkloadSelector, "80m", "230Mi", "900m", "900Mi")
		resourceassert.AssertIstioProxyResourcesEventually(t, c, httpbinRegularInfo.WorkloadSelector, "80m", "230Mi", "900m", "900Mi")
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

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		httpbinInfo, err := httpbin.NewBuilder().DeployWithCleanup(t)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"test-vs",
			defaultNamespace,
			httpbinInfo.Host,
			httpbinInfo.Host,
			kymaGateway,
		)
		require.NoError(t, err)

		gatewayAddress, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		hc := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("configuration-test"),
			httphelper.WithHost(httpbinInfo.Host),
			httphelper.WithHeaders(map[string]string{"X-Forwarded-For": "10.2.1.1,10.0.0.1"}),
		)
		url := fmt.Sprintf("http://%s/headers", gatewayAddress)
		httpbinassert.AssertHeaders(t, hc, url,
			httpbinassert.WithHeaderValue("X-Envoy-External-Address", "10.0.0.1"),
		)

		// when
		err = modulehelpers.NewIstioCRBuilder().
			WithName(istioCR.Name).
			WithNamespace(istioCR.Namespace).
			WithNumTrustedProxies(2).
			Update(t)
		require.NoError(t, err)

		// then
		httpbinassert.AssertHeaders(t, hc, url,
			httpbinassert.WithHeaderValue("X-Envoy-External-Address", "10.2.1.1"),
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

		istioassert.AssertEgressGatewayReady(t, c)
		require.NoError(t, err)
	})

	t.Run("External authorizer", func(t *testing.T) {
		c, err := client.ResourcesClient(t)
		require.NoError(t, err)

		err = infrahelpers.EnsureProductionClusterProfile(t)
		require.NoError(t, err)

		err = namespace.LabelNamespaceWithIstioInjection(t, defaultNamespace)
		require.NoError(t, err)

		extAuth, err := extauth.NewBuilder().WithName("ext-authz").WithNamespace(extAuthNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		extAuth2, err := extauth.NewBuilder().WithName("ext-authz2").WithNamespace(extAuthNamespace).DeployWithCleanup(t)
		require.NoError(t, err)

		authorizer1 := modulehelpers.NewAuthorizerFromExtAuth(extAuth)
		authorizer2 := modulehelpers.NewAuthorizerFromExtAuth(extAuth2)

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
			defaultNamespace,
			httpbinInfo.Host,
			httpbinInfo.Host,
			kymaGateway,
		)
		require.NoError(t, err)

		err = authzpolicy.CreateExtAuthzPolicy(t, "ext-authz", defaultNamespace, httpbinInfo.WorkloadSelector, "ext-authz", "/headers")
		require.NoError(t, err)

		httpbin2Info, err := httpbin.NewBuilder().WithName("httpbin-ext-auth2").DeployWithCleanup(t)
		require.NoError(t, err)

		err = virtualservice.CreateVirtualService(
			t,
			"httpbin-ext-auth2",
			defaultNamespace,
			httpbin2Info.Host,
			httpbin2Info.Host,
			gatewayhelper.GatewayReference,
		)
		require.NoError(t, err)

		err = gatewayhelper.CreateHTTPGateway(t)
		require.NoError(t, err)

		err = authzpolicy.CreateExtAuthzPolicy(t, "ext-authz2", defaultNamespace, httpbin2Info.WorkloadSelector, "ext-authz2", "/headers")
		require.NoError(t, err)

		gatewayAddress, err := load_balancer.GetLoadBalancerIP(t.Context(), c.GetControllerRuntimeClient())
		require.NoError(t, err)

		hc := httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("configuration-test"),
			httphelper.WithHost(httpbinInfo.Host),
		)
		url := fmt.Sprintf("http://%s/", gatewayAddress)
		httpassert.AssertOKResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("configuration-test"),
			httphelper.WithHost(httpbinInfo.Host),
			httphelper.WithHeaders(map[string]string{"x-ext-authz": "allow"}),
		)
		url = fmt.Sprintf("http://%s/headers", gatewayAddress)
		httpassert.AssertOKResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("configuration-test"),
			httphelper.WithHost(httpbinInfo.Host),
			httphelper.WithHeaders(map[string]string{"x-ext-authz": "deny"}),
		)
		httpassert.AssertForbiddenResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("configuration-test"),
			httphelper.WithHost(httpbin2Info.Host),
		)
		url = fmt.Sprintf("http://%s/", gatewayAddress)
		httpassert.AssertOKResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("configuration-test"),
			httphelper.WithHost(httpbin2Info.Host),
			httphelper.WithHeaders(map[string]string{"x-ext-authz": "allow"}),
		)
		url = fmt.Sprintf("http://%s/headers", gatewayAddress)
		httpassert.AssertOKResponse(t, hc, url)

		hc = httphelper.NewHTTPClient(t,
			httphelper.WithPrefix("configuration-test"),
			httphelper.WithHost(httpbin2Info.Host),
			httphelper.WithHeaders(map[string]string{"x-ext-authz": "deny"}),
		)
		httpassert.AssertForbiddenResponse(t, hc, url)
	})
}
