package install_istio

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/install_istio"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestInstallIstio(t *testing.T) {
	// Create executor
	e2eExecutor := executor.NewExecutor(t)
	defer e2eExecutor.Cleanup()

	// Install Istio
	steps := install_istio.Steps()
	for _, step := range steps {
		err := e2eExecutor.RunStep(step)
		require.NoError(t, err)
	}

	t.Log("Istio installation steps executed successfully")
}
