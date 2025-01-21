package predicates

import (
	v1 "k8s.io/api/core/v1"
)

type KymaWorkloadRestartPredicate struct {
}

func (p KymaWorkloadRestartPredicate) RequiresProxyRestart(pod v1.Pod) bool {
	return pod.Namespace == "kyma-system" || pod.Labels["kyma-project.io/module"] != ""
}
