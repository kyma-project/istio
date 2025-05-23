package predicates

import (
	v1 "k8s.io/api/core/v1"
)

type CustomerWorkloadRestartPredicate struct {
}

func NewCustomerWorkloadRestartPredicate() *CustomerWorkloadRestartPredicate {
	return &CustomerWorkloadRestartPredicate{}
}

func (p CustomerWorkloadRestartPredicate) Matches(pod v1.Pod) bool {
	return pod.Namespace != "kyma-system" && pod.Labels["kyma-project.io/module"] == ""
}

func (p CustomerWorkloadRestartPredicate) MustMatch() bool {
	return true
}
