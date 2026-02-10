package kubeconfig

import (
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"os"
	"testing"
)

func SwitchKubeConfig(t *testing.T, kubeconfigEnvVar string) error {
	t.Helper()
	beforeKubeconfig := os.Getenv("KUBECONFIG")
	newKubeconfig := os.Getenv(kubeconfigEnvVar)
	t.Logf("Switching KUBECONFIG from %s to %s", beforeKubeconfig, newKubeconfig)

	err := os.Setenv("KUBECONFIG", newKubeconfig)
	if err != nil {
		return err
	}

	clientSet, err := client.GetClientSet(t)
	if err != nil {
		return err
	}

	t.Logf("Current context after switching KUBECONFIG: %s", clientSet.RESTClient().Get().URL().String())

	setup.DeclareCleanup(t, func() {
		err := os.Setenv("KUBECONFIG", beforeKubeconfig)
		if err != nil {
			t.Logf("Failed to restore KUBECONFIG: %v", err)
		}
		t.Logf("Restored KUBECONFIG to %s", beforeKubeconfig)
		clientSet, err = client.GetClientSet(t)
		if err != nil {
			t.Logf("Failed to get client set after restoring KUBECONFIG: %v", err)
		}
		t.Logf("Current context after restoring KUBECONFIG: %s", clientSet.RESTClient().Get().URL().String())
	})
	return nil
}
