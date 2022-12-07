package restart

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func restartActionFactory(pod v1.Pod) restartAction {
	ownedBy, exists := getOwnerReferences(pod)

	if !exists {
		return newOwnerNotFoundAction(pod)
	}

	switch ownedBy.Kind {
	case "Job":
		return newOwnedByJobAction(pod)
	case "ReplicaSet":
	case "ReplicationController":
	default:
	}
	return nil

}

// getOwnerReferences returns the owner reference of the pod and a boolean to verify if the owner reference exists or not
func getOwnerReferences(pod v1.Pod) (*metav1.OwnerReference, bool) {
	if len(pod.OwnerReferences) == 0 {
		return &metav1.OwnerReference{}, false
	}

	return pod.OwnerReferences[0].DeepCopy(), true
}

type restartAction interface {
	run() ([]RestartWarning, error)
}

type actionObject struct {
	Name      string
	Namespace string
	Kind      string
}
