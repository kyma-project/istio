package egress_test

import (
	curlhelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/curl"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	modulehelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"

	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// TestE2EEgressConnectivity tests the connectivity of istio installed through istio-module.
// This test expects that istio-module is installed and access to cluster is set up via KUBECONFIG env.
func TestE2EEgressConnectivity(t *testing.T) {
	// initialization
	fsys := os.DirFS("testdata")

	// setup istio
	t.Log("Setting up Istio for the tests")
	require.NoError(t, modulehelpers.CreateIstioCR(t))

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
			runID := envconf.RandomName("e2e-egress", 16)
			// instantiate a resources client
			r := infrahelpers.ResourcesClient(t)

			// namespace creation
			require.NoError(t, infrahelpers.CreateNamespace(t, runID,
				infrahelpers.WithLabels(map[string]string{"istio-injection": "enabled"}),
			))
			if tt.applyEgressConfig {
				t.Log("Applying egress config for the test: ", runID)
				require.NoError(t, decoder.DecodeEachFile(t.Context(), fsys, "egress_config.yaml", decoder.CreateHandler(r), decoder.MutateNamespace(runID)))
			}
			if tt.applyNetworkPolicy {
				t.Log("Applying network policy for the test: ", runID)
				require.NoError(t, decoder.DecodeEachFile(t.Context(), fsys, "networkpolicy.yaml", decoder.CreateHandler(r), decoder.MutateNamespace(runID)))
			}

			// test using pod with curl
			err := curlhelper.RunCurlCmdInCluster(t, runID, tt.cmd)
			if err != nil && !tt.expectError {
				t.Errorf("got an error but shouldn't have: %v", err)
			}
			if err == nil && tt.expectError {
				t.Error("didn't get an error but expected one")
			}

		})
	}
}
