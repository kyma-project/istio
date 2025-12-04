package extauth

import (
	"context"
	_ "embed"
	"fmt"
	gatewayhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/gateway"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/portforward"
	"github.com/stretchr/testify/assert"
	"net/http"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/decoder"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/asserts/extauth"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/testsetup"
)

//go:embed ext_auth_virtual_service.yaml
var VirtualServiceExtAuth string

//go:embed ext_auth_istio_operator.yaml
var IstioExtAuthOperatorConfig string

//go:embed ext_auth_authorization_policy.yaml
var IstioExtAuthAuthorizationPolicy string

func TestAPIRuleExtAuth(t *testing.T) {
	require.NoError(t, infrahelpers.CreateNamespace(t, "ext-auth"))
	svcName, svcPort, err := httpbin.DeployHttpbin(t, "ext-auth")
	require.NoError(t, err, "Failed to deploy httpbin")

	t.Run("Timeout tests", func(t *testing.T) {
		t.Run("extauth set to /delay/1 should always return 200", func(t *testing.T) {
			// given
			require.NoError(
				t,
				modulehelpers.CreateIstioOperatorCR(t,
					modulehelpers.WithIstioOperatorTemplate(IstioExtAuthOperatorConfig),
					modulehelpers.WithIstioOperatorTemplateValues(map[string]interface{}{
						"ExtAuthServiceName":      svcName,
						"ExtAuthServicePort":      svcPort,
						"ExtAuthServiceNamespace": "ext-auth",
						"PathPrefix":              "/delay/1?original=",
						"Timeout":                 "2s",
					}),
				),
			)
			require.NoError(t, gatewayhelper.CreateHTTPGateway(t))
			gatewayDomain, gatewayPort, err := portforward.CreateIngressGatewayPortForwarding(t)
			require.NoError(t, err, "Failed to create port-forwarding to kyma-gateway")

			testBackground, err := testsetup.SetupRandomNamespaceWithOauth2MockAndHttpbin(t, testsetup.WithPrefix("extauth-test"))
			require.NoError(t, err, "Failed to setup test background with OAuth2 mock and httpbin")
			require.NoError(t, err, "Failed to deploy httpbin")

			// when
			createdVS, err := infrahelpers.CreateResourceWithTemplateValues(
				t,
				VirtualServiceExtAuth,
				map[string]any{
					"Name":            testBackground.TestName,
					"HostName":        fmt.Sprintf("%s.%s", testBackground.TestName, gatewayDomain),
					"DestinationHost": testBackground.TargetServiceName,
					"DestinationPort": testBackground.TargetServicePort,
					"GatewayName":     "kyma-system/kyma-gateway",
				},
				decoder.MutateNamespace(testBackground.Namespace),
			)
			require.NoError(t, err, "Failed to create VirtualService resource")
			require.NotEmpty(t, createdVS, "Created VirtualService resource should not be empty")

			createdAP, err := infrahelpers.CreateResource(
				t,
				IstioExtAuthAuthorizationPolicy,
				decoder.MutateNamespace(testBackground.Namespace),
			)
			require.NoError(t, err, "Failed to create AuthorizationPolicy resource")
			require.NotEmpty(t, createdAP, "Created AuthorizationPolicy resource should not be empty")

			// then
			waitFrom := time.Now()
			err = wait.For(func(ctx context.Context) (done bool, err error) {
				t.Logf("Waiting for endpoint to return 200 OK")
				t.Logf("Elapsed time: %v", time.Since(waitFrom))
				err = extauth.AssertEndpoint(t, http.MethodGet, fmt.Sprintf("http://%s.%s:%d/headers", testBackground.TestName, gatewayDomain, gatewayPort), map[string]string{}, http.StatusOK)
				if err != nil {
					return false, err
				}
				return true, nil
			}, wait.WithTimeout(10*time.Second), wait.WithInterval(1*time.Second))
			assert.NoError(t, err, "Failed to assert endpoint returns 200 OK")
		})

		t.Run("extauth set to /delay/3 should always return 403", func(t *testing.T) {
			// given
			require.NoError(
				t,
				modulehelpers.CreateIstioOperatorCR(t,
					modulehelpers.WithIstioOperatorTemplate(IstioExtAuthOperatorConfig),
					modulehelpers.WithIstioOperatorTemplateValues(map[string]interface{}{
						"ExtAuthServiceName":      svcName,
						"ExtAuthServicePort":      svcPort,
						"ExtAuthServiceNamespace": "ext-auth",
						"PathPrefix":              "/delay/3?original=",
						"Timeout":                 "2s",
					}),
				),
			)
			require.NoError(t, gatewayhelper.CreateHTTPGateway(t))
			gatewayDomain, gatewayPort, err := portforward.CreateIngressGatewayPortForwarding(t)
			require.NoError(t, err, "Failed to create port-forwarding to kyma-gateway")

			testBackground, err := testsetup.SetupRandomNamespaceWithOauth2MockAndHttpbin(t, testsetup.WithPrefix("extauth-test"))
			require.NoError(t, err, "Failed to setup test background with OAuth2 mock and httpbin")
			require.NoError(t, err, "Failed to deploy httpbin")

			// when
			createdVS, err := infrahelpers.CreateResourceWithTemplateValues(
				t,
				VirtualServiceExtAuth,
				map[string]any{
					"Name":            testBackground.TestName,
					"HostName":        fmt.Sprintf("%s.%s", testBackground.TestName, gatewayDomain),
					"DestinationHost": testBackground.TargetServiceName,
					"DestinationPort": testBackground.TargetServicePort,
					"GatewayName":     "kyma-system/kyma-gateway",
				},
				decoder.MutateNamespace(testBackground.Namespace),
			)
			require.NoError(t, err, "Failed to create VirtualService resource")
			require.NotEmpty(t, createdVS, "Created VirtualService resource should not be empty")

			createdAP, err := infrahelpers.CreateResource(
				t,
				IstioExtAuthAuthorizationPolicy,
				decoder.MutateNamespace(testBackground.Namespace),
			)
			require.NoError(t, err, "Failed to create AuthorizationPolicy resource")
			require.NotEmpty(t, createdAP, "Created AuthorizationPolicy resource should not be empty")

			// then
			waitFrom := time.Now()
			err = wait.For(func(ctx context.Context) (done bool, err error) {
				t.Logf("Waiting for endpoint to return 403 OK")
				t.Logf("Elapsed time: %v", time.Since(waitFrom))
				err = extauth.AssertEndpoint(t, http.MethodGet, fmt.Sprintf("http://%s.%s:%d/headers", testBackground.TestName, gatewayDomain, gatewayPort), map[string]string{}, http.StatusForbidden)
				if err != nil {
					return false, err
				}
				return true, nil
			}, wait.WithTimeout(10*time.Second), wait.WithInterval(1*time.Second))
			assert.NoError(t, err, "Failed to assert endpoint returns 403 Forbidden")
		})
	})

	t.Run("Status tests", func(t *testing.T) {
		t.Run("extauth set to /status/200 should always return 200", func(t *testing.T) {
			// given
			require.NoError(
				t,
				modulehelpers.CreateIstioOperatorCR(t,
					modulehelpers.WithIstioOperatorTemplate(IstioExtAuthOperatorConfig),
					modulehelpers.WithIstioOperatorTemplateValues(map[string]interface{}{
						"ExtAuthServiceName":      svcName,
						"ExtAuthServicePort":      svcPort,
						"ExtAuthServiceNamespace": "ext-auth",
						"PathPrefix":              "/status/200?original=",
						"Timeout":                 "5s",
					}),
				),
			)
			require.NoError(t, gatewayhelper.CreateHTTPGateway(t))
			testBackground, err := testsetup.SetupRandomNamespaceWithOauth2MockAndHttpbin(t, testsetup.WithPrefix("extauth-test"))
			require.NoError(t, err, "Failed to setup test background with OAuth2 mock and httpbin")

			gatewayDomain, gatewayPort, err := portforward.CreateIngressGatewayPortForwarding(t)
			require.NoError(t, err, "Failed to create port-forwarding to kyma-gateway")

			// when
			createdVS, err := infrahelpers.CreateResourceWithTemplateValues(
				t,
				VirtualServiceExtAuth,
				map[string]any{
					"Name":            testBackground.TestName,
					"HostName":        fmt.Sprintf("%s.%s", testBackground.TestName, gatewayDomain),
					"DestinationHost": testBackground.TargetServiceName,
					"DestinationPort": testBackground.TargetServicePort,
					"GatewayName":     "kyma-system/kyma-gateway",
				},
				decoder.MutateNamespace(testBackground.Namespace),
			)
			require.NoError(t, err, "Failed to create VirtualService resource")
			require.NotEmpty(t, createdVS, "Created VirtualService resource should not be empty")

			createdAP, err := infrahelpers.CreateResource(
				t,
				IstioExtAuthAuthorizationPolicy,
				decoder.MutateNamespace(testBackground.Namespace),
			)
			require.NoError(t, err, "Failed to create AuthorizationPolicy resource")
			require.NotEmpty(t, createdAP, "Created AuthorizationPolicy resource should not be empty")

			// then
			waitFrom := time.Now()
			err = wait.For(func(ctx context.Context) (done bool, err error) {
				t.Logf("Waiting for endpoint to return 200 OK")
				t.Logf("Elapsed time: %v", time.Since(waitFrom))
				err = extauth.AssertEndpoint(t, http.MethodGet, fmt.Sprintf("http://%s.%s:%d/headers", testBackground.TestName, gatewayDomain, gatewayPort), map[string]string{}, http.StatusOK)
				if err != nil {
					return false, err
				}
				return true, nil
			}, wait.WithTimeout(10*time.Second), wait.WithInterval(1*time.Second))
			assert.NoError(t, err, "Failed to assert endpoint returns 200 OK")
		})

		t.Run("extauth set to /status/403 should always return 403", func(t *testing.T) {
			// given
			require.NoError(
				t,
				modulehelpers.CreateIstioOperatorCR(t,
					modulehelpers.WithIstioOperatorTemplate(IstioExtAuthOperatorConfig),
					modulehelpers.WithIstioOperatorTemplateValues(map[string]interface{}{
						"ExtAuthServiceName":      svcName,
						"ExtAuthServicePort":      svcPort,
						"ExtAuthServiceNamespace": "ext-auth",
						"PathPrefix":              "/status/403?original=",
						"Timeout":                 "5s",
					}),
				),
			)
			require.NoError(t, gatewayhelper.CreateHTTPGateway(t))
			gatewayDomain, gatewayPort, err := portforward.CreateIngressGatewayPortForwarding(t)
			require.NoError(t, err, "Failed to create port-forwarding to kyma-gateway")

			testBackground, err := testsetup.SetupRandomNamespaceWithOauth2MockAndHttpbin(t, testsetup.WithPrefix("extauth-test"))
			require.NoError(t, err, "Failed to setup test background with OAuth2 mock and httpbin")
			require.NoError(t, err, "Failed to deploy httpbin")

			// when
			createdVS, err := infrahelpers.CreateResourceWithTemplateValues(
				t,
				VirtualServiceExtAuth,
				map[string]any{
					"Name":            testBackground.TestName,
					"HostName":        fmt.Sprintf("%s.%s", testBackground.TestName, gatewayDomain),
					"DestinationHost": testBackground.TargetServiceName,
					"DestinationPort": testBackground.TargetServicePort,
					"GatewayName":     "kyma-system/kyma-gateway",
				},
				decoder.MutateNamespace(testBackground.Namespace),
			)
			require.NoError(t, err, "Failed to create VirtualService resource")
			require.NotEmpty(t, createdVS, "Created VirtualService resource should not be empty")

			createdAP, err := infrahelpers.CreateResource(
				t,
				IstioExtAuthAuthorizationPolicy,
				decoder.MutateNamespace(testBackground.Namespace),
			)
			require.NoError(t, err, "Failed to create AuthorizationPolicy resource")
			require.NotEmpty(t, createdAP, "Created AuthorizationPolicy resource should not be empty")

			// then
			waitFrom := time.Now()
			err = wait.For(func(ctx context.Context) (done bool, err error) {
				t.Logf("Waiting for endpoint to return 403 Forbidden: %v", err)

				t.Logf("Elapsed time: %v", time.Since(waitFrom))
				err = extauth.AssertEndpoint(t, http.MethodGet, fmt.Sprintf("http://%s.%s:%d/headers", testBackground.TestName, gatewayDomain, gatewayPort), map[string]string{}, http.StatusForbidden)
				if err != nil {
					return false, err
				}
				return true, nil
			}, wait.WithTimeout(10*time.Second), wait.WithInterval(1*time.Second))
			assert.NoError(t, err, "Failed to assert endpoint returns 403 Forbidden")
		})
	})
}
