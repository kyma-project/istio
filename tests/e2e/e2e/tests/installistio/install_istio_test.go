package installistio_test

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	unstructuredstep "github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/infrastructure/unstructured"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"
	"time"

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

		require.Eventually(t, func() bool {
			err := e2eExecutor.RunStep(getIstiodDeployment)
			if err != nil {
				logging.Errorf(t, "Failed to get Istiod deployment: %v", err)
				return false
			}
			return getIstiodDeployment.Output != nil
		}, 30*time.Second, 5*time.Second, "Istiod deployment should be available after installation")

		retrievedDeployment := getIstiodDeployment.Output
		require.NotNil(t, retrievedDeployment, "Expected a non-nil Istiod pod after installation")

		logging.Debugf(t, "Installed istio pilot: %+v", retrievedDeployment.Object)
		logging.Infof(t, "Istio installation steps executed successfully")
	})
}
