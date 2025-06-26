package createnamespacewithkubectl_test

import (
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/helpers/kubectl/namespace"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateNsWithKubectl(t *testing.T) {
	testId := setup.GenerateRandomTestId()
	namespaceName := fmt.Sprintf("test-ns-%s", testId)

	t.Run("Create Namespace", func(t *testing.T) {
		t.Parallel()

		output, err := namespace.CreateNamespace(t, namespaceName)
		require.NoError(t, err)
		require.Contains(t,
			string(output),
			fmt.Sprintf("namespace/%s created", namespaceName),
			"Expected namespace %s creation confirmation in output",
			namespaceName,
		)

		// Verify Namespace Creation
		output, err = namespace.GetNamespace(t, namespaceName)
		t.Logf("Get namespace output: %s", string(output))
		require.NoError(t, err, "Namespace should be fetched successfully")
		require.Contains(t,
			string(output),
			namespaceName,
			"Expected namespace %s to be present in the output",
			namespaceName)
		t.Logf("Namespace created successfully: %s", output)
	})
}
