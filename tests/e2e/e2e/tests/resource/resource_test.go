package resource_test

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/infrastructure/yamlfile"
)

func TestCreateResource(t *testing.T) {
	// Create executor
	e2eExecutor := executor.NewExecutorWithOptionsFromEnv(t)
	defer e2eExecutor.Cleanup()

	// given
	createResource := yamlfile.Create{FilePath: "pod.yaml"}
	err := e2eExecutor.RunStep(&createResource)
	require.NoError(t, err)

	// when
	getResource := yamlfile.Get{FilePath: "pod.yaml"}
	err = e2eExecutor.RunStep(&getResource)
	require.NoError(t, err)

	// then
	retrievedResource := getResource.Output()
	require.NotNil(t, retrievedResource, "Expected a non-nil resource after creation")
	logging.Debugf(t, "Created resource: %s", retrievedResource)
}
