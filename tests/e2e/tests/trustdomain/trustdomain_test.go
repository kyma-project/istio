package trustdomain

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpbin"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/httpincluster"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
	"github.com/stretchr/testify/require"
)

//go:embed istio-operator.yaml
var IstioOperator string

func TestTrustDomain(t *testing.T) {
	t.Run("Test trust domain", func(t *testing.T) {
		t.Run("mTLS between meshes should work with origination from sidecar", func(t *testing.T) {
			require.NoError(
				t,
				modulehelpers.CreateIstioOperatorCR(t,
					modulehelpers.WithIstioOperatorTemplate(IstioOperator),
				),
			)

			require.NoError(t, infrahelpers.CreateNamespace(t, "httpbin", infrahelpers.WithSidecarInjectionEnabled()))
			svcName, svcPort, err := httpbin.DeployHttpbin(t, "httpbin")
			require.NoError(t, err)

			require.NoError(t, infrahelpers.CreateNamespace(t, "verifier"))
			stdOut, stdErr, _ := httpincluster.RunOpenSSLSClientFromInsideCluster(t, "verifier",
				fmt.Sprintf("%s.httpbin.svc.cluster.local:%d", svcName, svcPort))
			t.Logf("StdOut: %s", stdOut)
			t.Logf("StdErr: %s", stdErr)

			require.Contains(t, stdOut, "Acceptable client certificate CA names\nO=client.trust.domain")
		})
	})

}
