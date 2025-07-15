package egress_test

import (
	"bytes"
	_ "embed"
	curlhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/curl"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"

	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

//go:embed istio_cr_with_egress.yaml
var istioCRWithEgress []byte

//go:embed egress_config.yaml
var egressConfig []byte

//go:embed networkpolicy.yaml
var networkPolicy []byte

// TestE2EEgressConnectivity tests the connectivity of istio installed through istio-module.
// This test expects that istio-module is installed and access to cluster is set up via KUBECONFIG env.
func TestE2EEgressConnectivity(t *testing.T) {
	// setup istio
	t.Log("Setting up Istio for the tests")
	require.NoError(t, modulehelpers.CreateIstioCR(t, modulehelpers.IstioCROptions{
		Template: string(istioCRWithEgress),
	}))

	// initialize testcases
	// note: test might fail randomly from random downtime to httpbin.org with error Connection reset by peer.
	// This is a flake, and we need to think how to resolve that eventually.
	tc := []struct {
		name                 string
		cmd                  []string
		expectError          bool
		applyNetworkPolicy   bool
		applyEgressConfig    bool
		enableIstioInjection bool
	}{
		{
			name:              "connection to httpbin service is OK when egress is deployed",
			cmd:               []string{"curl", "-sSL", "-m", "10", "https://httpbin.org/headers"},
			applyEgressConfig: true,
			expectError:       false,
		},
		{
			name:               "connection to httpbin service is refused when NetworkPolicy is applied",
			cmd:                []string{"curl", "-sSL", "-m", "10", "https://httpbin.org/headers"},
			applyNetworkPolicy: true,
			// sidecar init fails when NP is applied. When uncommented, the test will pass despite confirming manually
			// that connection is refused with NP
			expectError: true,
		},
		{
			name:               "connection to httpbin service is OK when NetworkPolicy is applied and egress is configured",
			cmd:                []string{"curl", "-sSL", "-m", "10", "https://httpbin.org/headers"},
			applyEgressConfig:  true,
			applyNetworkPolicy: true,
			expectError:        false,
		},
		{
			name:               "connection to kyma-project is refused when NetworkPolicy is applied and egress is configured",
			cmd:                []string{"curl", "-sSL", "-m", "10", "https://kyma-project.io"},
			applyEgressConfig:  true,
			applyNetworkPolicy: true,
			expectError:        true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			runNamespace := envconf.RandomName("e2e-egress", 16)
			// instantiate a resources client
			r := infrahelpers.ResourcesClient(t)

			// namespace creation
			require.NoError(t, infrahelpers.CreateNamespace(t, runNamespace,
				infrahelpers.WithLabels(map[string]string{"istio-injection": "enabled"}),
			))
			if tt.applyEgressConfig {
				t.Logf("Applying egress config for the test in namespace %s", runNamespace)
				require.NoError(t,
					decoder.DecodeEach(
						t.Context(),
						bytes.NewBuffer(egressConfig),
						decoder.CreateHandler(r),
						decoder.MutateNamespace(runNamespace),
					),
				)
			}
			if tt.applyNetworkPolicy {
				t.Logf("Applying network policy for the test in namespace %s", runNamespace)
				require.NoError(t,
					decoder.DecodeEach(
						t.Context(),
						bytes.NewBuffer(networkPolicy),
						decoder.CreateHandler(r),
						decoder.MutateNamespace(runNamespace),
					),
				)
			}

			// test using pod with curl
			err := curlhelper.RunCurlCmdInCluster(t, runNamespace, tt.cmd)
			if err != nil && !tt.expectError {
				t.Errorf("got an error but shouldn't have: %v", err)
			}
			if err == nil && tt.expectError {
				t.Error("didn't get an error but expected one")
			}

		})
	}
}
