package newtest_test

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/installistio"
	"testing"
)

// Struktura opakowania testów
// type TestSuite struct {
// given Steps[]
// when Steps[]
// then Steps[]

func TestAPsInIstio(t *testing.T) {
	// Tutaj setup environmentu
	setupExecutor := executor.NewExecutorWithOptionsFromEnv(t)
	defer setupExecutor.Cleanup()

	//setupSteps
	installIstioSteps := installistio.Steps()
	for _, step := range installIstioSteps {
		err := setupExecutor.RunStep(step)
		if err != nil {
			t.Fatalf("Failed to run step %s: %v", step.Description(), err)
		}
	}

	// Tutaj testy:
	t.Run("Check istiod", func(t *testing.T) {
		testExecutor := executor.NewExecutorWithOptionsFromEnv(t)
		defer testExecutor.Cleanup()

	})

	t.Run("Check ingress-gateway", func(t *testing.T) {
		testExecutor := executor.NewExecutorWithOptionsFromEnv(t)
		defer testExecutor.Cleanup()

	})
}
