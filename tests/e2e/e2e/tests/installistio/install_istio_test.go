package installistio_test

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	unstructuredstep "github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/infrastructure/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/installistio"
)

func TestInstallIstio(t *testing.T) {
	t.Run("Install Istio", func(t *testing.T) {
		e2eExecutor := executor.NewExecutorWithOptionsFromEnv(t)
		defer e2eExecutor.Cleanup()

		// given
		steps := installistio.Steps()

		// when
		for _, step := range steps {
			err := e2eExecutor.RunStep(step)
			require.NoError(t, err)
		}

		// then
		getIstiodDeployment := &unstructuredstep.Get{
			Namespace: "istio-system",
			Name:      "istiod",
			GVK: schema.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    "Deployment",
			},
		}
		err := e2eExecutor.RunStep(getIstiodDeployment, executor.RunStepOptions{
			RetryPeriod: 5 * time.Second,
			Timeout:     1 * time.Minute,
		})
		require.NoError(t, err)
		retrievedDeployment := getIstiodDeployment.Output()
		require.NotNil(t, retrievedDeployment, "Expected a non-nil Istiod pod after installation")

		logging.Debugf(t, "Installed istio pilot: %+v", retrievedDeployment.Object)
		logging.Infof(t, "Istio installation steps executed successfully")
	})
}
