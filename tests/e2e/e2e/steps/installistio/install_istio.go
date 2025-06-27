package installistio

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/exec"
)

func Steps() []executor.Step {
	createNamespaceStep := &exec.Command{
		Command:     "kubectl",
		Args:        []string{"create", "namespace", "kyma-system"},
		CleanupCmd:  "kubectl",
		CleanupArgs: []string{"delete", "namespace", "kyma-system"},
	}

	createManagerStep := &exec.Command{
		Command:     "kubectl",
		Args:        []string{"apply", "-f", "https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml"},
		CleanupCmd:  "kubectl",
		CleanupArgs: []string{"delete", "-f", "https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml"},
	}

	createIstioCRStep := &exec.Command{
		Command:     "kubectl",
		Args:        []string{"apply", "-f", "https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml"},
		CleanupCmd:  "kubectl",
		CleanupArgs: []string{"delete", "-f", "https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml"},
	}

	return []executor.Step{
		createNamespaceStep,
		createManagerStep,
		createIstioCRStep,
	}
}
