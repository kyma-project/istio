package restart

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type rolloutAction struct {
	object actionObject
}

func newRolloutAction(pod v1.Pod) rolloutAction {
	return rolloutAction{}
}

func newRolloutActionObject(pod v1.Pod, ownedBy *metav1.OwnerReference) actionObject {
	return actionObject{
		Name:      ownedBy.Name,
		Namespace: pod.Namespace,
		Kind:      ownedBy.Kind,
	}
}
