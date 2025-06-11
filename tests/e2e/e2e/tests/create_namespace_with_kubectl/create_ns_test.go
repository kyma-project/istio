package create_namespace_with_kubectl

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	bashStep "github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/exec"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateNsWithKubectl(t *testing.T) {
	t.Parallel()

	t.Run("Create Namespace", func(t *testing.T) {
		// Setup Infra
		testExecutor := executor.NewExecutor(t)
		defer testExecutor.Cleanup()

		createNs := &bashStep.Command{
			Command:    "kubectl create namespace test-namespace",
			CleanupCmd: "kubectl delete namespace test-namespace",
		}

		err := testExecutor.RunStep(createNs)
		require.NoError(t, err)

		// Verify Namespace Creation
		verifyNs := &bashStep.Command{
			Command: "kubectl get namespace test-namespace --no-headers -o custom-columns=NAME:.metadata.name",
		}
		err = testExecutor.RunStep(verifyNs)
		require.NoError(t, err, "Namespace should be created successfully")
		output := verifyNs.Output
		require.Contains(t, string(output), "test-namespace", "Expected namespace 'test-namespace' to be present in the output")
		executor.Debugf(t, "Namespace created successfully: %s", output)
	})
}
