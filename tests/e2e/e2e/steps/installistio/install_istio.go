package installistio

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/exec"
)

func Steps() []executor.Step {
	createNamespaceStep := &exec.Command{
		Command:    "kubectl create namespace kyma-system",
		CleanupCmd: "kubectl delete namespace kyma-system",
	}

	createManagerStep := &exec.Command{
		Command:    "kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml",
		CleanupCmd: "kubectl delete -f https://github.com/kyma-project/istio/releases/latest/download/istio-manager.yaml",
	}

	createIstioCRStep := &exec.Command{
		Command:    "kubectl apply -f https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml",
		CleanupCmd: "kubectl delete -f https://github.com/kyma-project/istio/releases/latest/download/istio-default-cr.yaml",
	}

	return []executor.Step{
		createNamespaceStep,
		createManagerStep,
		createIstioCRStep,
	}
}
