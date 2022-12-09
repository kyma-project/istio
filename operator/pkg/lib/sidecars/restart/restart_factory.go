package restart

import (
	"context"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func restartActionFactory(ctx context.Context, c client.Client, pod v1.Pod) restartAction {
	ownedBy, exists := getOwnerReferences(pod)

	if !exists {
		return newOwnerNotFoundAction(pod)
	}

	switch ownedBy.Kind {
	case "Job":
		return newOwnedByJobAction(pod)
	case "ReplicaSet":
		return getReplicaSetAction(ctx, c, pod, ownedBy)
	case "ReplicationController":
		// TODO: Decide on action in here
		fallthrough
	default:
		return newRolloutAction(actionObject{
			Name:      ownedBy.Name,
			Namespace: pod.Namespace,
			Kind:      ownedBy.Kind,
		})
	}
}

// getOwnerReferences returns the owner reference of the pod and a boolean to verify if the owner reference exists or not
func getOwnerReferences(pod v1.Pod) (*metav1.OwnerReference, bool) {
	if len(pod.OwnerReferences) == 0 {
		return &metav1.OwnerReference{}, false
	}

	return pod.OwnerReferences[0].DeepCopy(), true
}

type restartAction struct {
	run    func(context.Context, client.Client, actionObject) ([]RestartWarning, error)
	object actionObject
}

type actionObject struct {
	Name      string
	Namespace string
	Kind      string
}

func actionObjectFromPod(pod v1.Pod) actionObject {
	return actionObject{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Kind:      pod.Kind,
	}
}
