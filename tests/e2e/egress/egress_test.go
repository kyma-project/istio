package egress_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"testing"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/ns"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

// TestE2EEgressConnectivity tests the connectivity of istio installed through istio-module.
// This test expects that istio-module is installed and access to cluster is set up via KUBECONFIG env.
func TestE2EEgressConnectivity(t *testing.T) {
	// initialization
	// we cannot use t.Context because it is cancelled just after the test function finishes.
	// it will be already cancelled when running cleanup functions, making cleanup not possible
	ctx := context.Background()
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	fsys := os.DirFS("testdata")

	// setup istio
	t.Log("Setting up Istio for the tests")
	require.NoError(t, setupIstio(ctx, fsys, cfg, t))

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
			cmd:               []string{"curl", "-sSL", "-m", "10", "--fail-with-body", "https://httpbin.org/headers"},
			applyEgressConfig: true,
			expectError:       false,
		},
		{
			name:               "connection to httpbin service is refused when NetworkPolicy is applied",
			cmd:                []string{"curl", "-sSL", "-m", "10", "--fail-with-body", "https://httpbin.org/headers"},
			applyNetworkPolicy: true,
			// sidecar init fails when NP is applied. When uncommented, the test will pass despite confirming manually
			// that connection is refused with NP
			expectError: true,
		},
		{
			name:               "connection to httpbin service is OK when NetworkPolicy is applied and egress is configured",
			cmd:                []string{"curl", "-sSL", "-m", "10", "--fail-with-body", "https://httpbin.org/headers"},
			applyEgressConfig:  true,
			applyNetworkPolicy: true,
			expectError:        false,
		},
		{
			name:               "connection to kyma-project is refused when NetworkPolicy is applied and egress is configured",
			cmd:                []string{"curl", "-sSL", "-m", "10", "--fail-with-body", "https://kyma-project.io"},
			applyEgressConfig:  true,
			applyNetworkPolicy: true,
			expectError:        true,
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			runID := envconf.RandomName("e2e-egress", 16)
			// instantiate a resources client
			r, tErr := resources.New(cfg.Client().RESTConfig())
			require.NoError(t, tErr)

			// namespace creation
			require.NoError(t, ns.CreateNamespace(ctx, t, runID, cfg,
				ns.WithLabels(map[string]string{"istio-injection": "enabled"}),
			))
			if tt.applyEgressConfig {
				t.Log("Applying egress config for the test: ", runID)
				require.NoError(t, decoder.DecodeEachFile(ctx, fsys, "egress_config.yaml", decoder.CreateHandler(r), decoder.MutateNamespace(runID)))
			}
			if tt.applyNetworkPolicy {
				t.Log("Applying network policy for the test: ", runID)
				require.NoError(t, decoder.DecodeEachFile(ctx, fsys, "networkpolicy.yaml", decoder.CreateHandler(r), decoder.MutateNamespace(runID)))
			}

			// test using pod with curl
			testPod := &corev1.Pod{}
			require.NoError(t, decoder.DecodeFile(fsys, "curl_pod.yaml", testPod, decoder.MutateNamespace(runID)))
			require.NoError(t, r.Create(ctx, testPod))
			require.NoError(t, wait.For(conditions.New(r).PodRunning(testPod)))

			var stdout, stderr bytes.Buffer
			err := r.ExecInPod(ctx, runID, testPod.Name, "curl", tt.cmd, &stdout, &stderr)
			if err != nil && !tt.expectError {
				t.Errorf("got an error but shouldn't have: %v", err)
			}
			if err == nil && tt.expectError {
				t.Error("didn't get an error but expected one")
			}

			t.Log(stdout.String())
			t.Log(stderr.String())
		})
	}
}

func setupIstio(ctx context.Context, fsys fs.FS, cfg *envconf.Config, t *testing.T) error {
	t.Helper()
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return err
	}
	_ = v1alpha2.AddToScheme(r.GetScheme())

	icr := &v1alpha2.Istio{}
	err = decoder.DecodeFile(fsys, "istio_customresource.yaml", icr)
	if err != nil {
		return err
	}
	err = r.Create(ctx, icr)
	if err != nil {
		return err
	}

	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up Istio after the tests")
		require.NoError(t, teardownIstio(ctx, fsys, cfg, t))
	})
	// Wait for Istio to be ready
	err = wait.For(conditions.New(r).ResourceMatch(icr, func(obj k8s.Object) bool {
		icrObj, ok := obj.(*v1alpha2.Istio)
		if !ok {
			return false
		}
		return icrObj.Status.State == v1alpha2.Ready
	}))

	return nil
}
func teardownIstio(ctx context.Context, fsys fs.FS, cfg *envconf.Config, t *testing.T) error {
	t.Helper()
	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return err
	}
	_ = v1alpha2.AddToScheme(r.GetScheme())

	icr := &v1alpha2.Istio{}
	err = decoder.DecodeFile(fsys, "istio_customresource.yaml", icr, decoder.MutateNamespace("kyma-system"))
	if err != nil {
		return err
	}

	err = r.Delete(ctx, icr)
	if err != nil {
		return err
	}

	err = wait.For(conditions.New(r).ResourceDeleted(icr))
	if err != nil {
		return err
	}

	return nil
}
