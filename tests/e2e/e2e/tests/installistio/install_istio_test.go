package installistio_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/installistio"
)

func TestInstallIstio(t *testing.T) {
	// Create executor
	e2eExecutor := executor.NewExecutorWithOptionsFromEnv(t)
	defer e2eExecutor.Cleanup()

	// Install Istio
	steps := installistio.Steps()
	for _, step := range steps {
		err := e2eExecutor.RunStep(step)
		require.NoError(t, err)
	}

	t.Log("Istio installation steps executed successfully")
}
