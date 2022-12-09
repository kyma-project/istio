package restart

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const rolloutTimeoutMessage = "pod could not be rolled out by resource owner's controller."

type rolloutAction struct {
	object actionObject
}

func newRolloutAction(pod v1.Pod, ownedBy *metav1.OwnerReference) rolloutAction {
	return rolloutAction{
		object: actionObject{
			Name:      ownedBy.Name,
			Namespace: pod.Namespace,
			Kind:      ownedBy.Kind,
		},
	}
}

func (r rolloutAction) run() ([]RestartWarning, error) {
	return []RestartWarning{newRestartWarning(r.object, rolloutTimeoutMessage)}, nil
}
