package namespace

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/helpers/shell"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"testing"
)

func CreateNamespace(t *testing.T, name string) ([]byte, error) {
	t.Helper()

	setup.DeclareCleanup(t, func() {
		t.Logf("Deleting namespace: name: %s", name)
		output, err := shell.Execute(t, "kubectl", "delete", "namespace", name)
		if err != nil {
			t.Logf("Error deleting namespace: name: %s, output: %s, error: %s", name, string(output), err)
		}
	})

	t.Logf("Creating Namespace: name: %s", name)
	return shell.Execute(t, "kubectl", "create", "namespace", name)
}

func GetNamespace(t *testing.T, name string) ([]byte, error) {
	t.Helper()

	t.Logf("Getting namespace: name: %s", name)
	return shell.Execute(t, "kubectl", "get", "namespace", name, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
}
