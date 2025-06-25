package createnamespacewithkubectl_test

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	bashStep "github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/exec"
)

func TestCreateNsWithKubectl(t *testing.T) {
	t.Parallel()

	t.Run("Create Namespace", func(t *testing.T) {
		t.Parallel()
		// Setup Infra
		testExecutor := executor.NewExecutorWithOptionsFromEnv(t)
		defer testExecutor.Cleanup()

		createNs := &bashStep.Command{
			Command:     "kubectl",
			Args:        []string{"create", "namespace", "test-namespace"},
			CleanupCmd:  "kubectl",
			CleanupArgs: []string{"delete", "namespace", "test-namespace"},
		}

		err := testExecutor.RunStep(createNs)
		output, exitCode := createNs.Output, createNs.ExitCode

		require.NoError(t, err)
		require.Equal(t, 0, exitCode)
		require.Contains(t, string(output), "namespace/test-namespace created", "Expected namespace creation confirmation in output")

		// Verify Namespace Creation
		verifyNs := &bashStep.Command{
			Command: "kubectl",
			Args:    []string{"get", "namespace", "test-namespace", "--no-headers", "-o", "custom-columns=NAME:.metadata.name"},
		}
		err = testExecutor.RunStep(verifyNs)
		require.NoError(t, err, "Namespace should be fetched successfully")

		output, exitCode = verifyNs.Output, verifyNs.ExitCode
		require.Equal(t, 0, exitCode)
		require.Contains(t, string(output), "test-namespace", "Expected namespace 'test-namespace' to be present in the output")
		logging.Debugf(t, "Namespace created successfully: %s", output)
	})
}
