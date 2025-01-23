package predicates

import (
	v1 "k8s.io/api/core/v1"
)

type KymaWorkloadRestartPredicate struct {
}

func (p KymaWorkloadRestartPredicate) Matches(pod v1.Pod) bool {
	return pod.Namespace == "kyma-system" || pod.Labels["kyma-project.io/module"] != ""
}

func (p KymaWorkloadRestartPredicate) MustMatch() bool {
	return true
}

func NewKymaWorkloadRestartPredicate() *KymaWorkloadRestartPredicate {
	return &KymaWorkloadRestartPredicate{}
}
