package trustdomain

import (
	_ "embed"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/kubeconfig"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"testing"
)

// ### CLIENT CLUSTER MANIFESTS ###

//go:embed client-cluster-secret.yaml
var ClientClusterSecret string

//go:embed client-manifests/istio-operator.yaml
var ClientIstioOperator string

//go:embed client-manifests/service-entry-sidecar.yaml
var ServiceEntrySidecar string

//go:embed client-manifests/destination-rule-sidecar.yaml
var DestinationRuleSidecar string

// ### SERVER CLUSTER MANIFESTS ###

//go:embed server-cluster-secret.yaml
var ServerClusterSecret string

//go:embed server-manifests/istio-operator.yaml
var ServerIstioOperator string

//go:embed server-manifests/gateway.yaml
var ServerGateway string

//go:embed server-manifests/virtual-service.yaml
var ServerVirtualService string

func TestTrustDomain(t *testing.T) {
	t.Run("Test trust domain", func(t *testing.T) {
		t.Run("mTLS between meshes should work with origination from sidecar", func(t *testing.T) {
			// ### SERVER CLUSTER SETUP ###
			require.NoError(t, kubeconfig.SwitchKubeConfig(t, "KUBECONFIG2"))

			createdSecret, err := infrastructure.CreateResource(t,
				ServerClusterSecret,
			)
			require.NoError(t, err)
			t.Logf("Created secret %s/%s in server cluster", createdSecret.GetNamespace(), createdSecret.GetName())

			require.NoError(
				t,
				modulehelpers.CreateIstioOperatorCR(t,
					modulehelpers.WithIstioOperatorTemplate(ServerIstioOperator),
				),
			)

			require.NoError(t, infrahelpers.CreateNamespace(t, "trust-domain", infrahelpers.WithSidecarInjectionEnabled()))
			svcName, svcPort, err := httpbin.DeployHttpbin(t, "trust-domain")
			require.NoError(t, err)
			t.Logf("Deployed httpbin in server cluster with service %s and port %d", svcName, svcPort)

			_, err = infrahelpers.CreateResource(t,
				ServerGateway,
				decoder.MutateNamespace("trust-domain"),
			)
			require.NoError(t, err)

			_, err = infrahelpers.CreateResourceWithTemplateValues(t,
				ServerVirtualService,
				map[string]any{
					"SERVICE_NAME": svcName,
					"SERVICE_PORT": svcPort,
				},
				decoder.MutateNamespace("trust-domain"),
			)
			require.NoError(t, err)

			// ### CLIENT CLUSTER SETUP ###
			require.NoError(t, kubeconfig.SwitchKubeConfig(t, "KUBECONFIG1"))

			require.NoError(t, infrastructure.CreateNamespace(t,
				"istio-system",
			))

			createdSecret, err = infrastructure.CreateResource(t,
				ClientClusterSecret,
			)
			require.NoError(t, err)
			t.Logf("Created secret %s/%s in client cluster", createdSecret.GetNamespace(), createdSecret.GetName())

			require.NoError(
				t,
				modulehelpers.CreateIstioOperatorCR(t,
					modulehelpers.WithIstioOperatorTemplate(ClientIstioOperator),
				),
			)

			// ### TESTING (CLIENT CLUSTER) ###
			require.NoError(t, infrahelpers.CreateNamespace(t, "external-httpbin", infrahelpers.WithSidecarInjectionEnabled()))
			_, err = infrahelpers.CreateResourceWithTemplateValues(t,
				ServiceEntrySidecar,
				map[string]any{
					"WORKLOAD_DOMAIN": serverClusterDomain,
				},
			)
			require.NoError(t, err)

			_, err = infrahelpers.CreateResourceWithTemplateValues(t,
				DestinationRuleSidecar,
				map[string]any{
					"WORKLOAD_DOMAIN": serverClusterDomain,
				},
			)
			require.NoError(t, err)
		})
	})

}
