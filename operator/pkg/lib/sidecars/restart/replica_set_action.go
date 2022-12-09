package restart

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getReplicaSetAction(ctx context.Context, c client.Client, pod v1.Pod, replicaSetRef *metav1.OwnerReference) restartAction {

	replicaSetKey := client.ObjectKey{
		Namespace: pod.Namespace,
		Name:      replicaSetRef.Name,
	}

	var replicaSet = &appsv1.ReplicaSet{}
	err := c.Get(ctx, replicaSetKey, replicaSet)
	if err != nil {
		// TODO in the existing code only logging happens and execution continues. Is this a good thing?
	}

	// TODO for better understanding - why do we delete the RS if there is no parent?
	if rsOwnedBy, exists := getReplicaSetOwner(replicaSet); !exists {
		// TODO add pod delete action
		return nil
	} else {
		return rolloutAction{
			object: actionObject{
				Name:      rsOwnedBy.Name,
				Namespace: replicaSet.Namespace,
				Kind:      rsOwnedBy.Kind,
			},
		}
	}
}

// getOwnerReferences returns the owner reference of the pod and a boolean to verify if the owner reference exists or not
func getReplicaSetOwner(rs *appsv1.ReplicaSet) (*metav1.OwnerReference, bool) {
	if len(rs.OwnerReferences) == 0 {
		return &metav1.OwnerReference{}, false
	}

	return rs.OwnerReferences[0].DeepCopy(), true
}
