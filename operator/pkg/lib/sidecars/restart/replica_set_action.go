package restart

import (
	"context"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func getReplicaSetAction(ctx context.Context, c client.Client, pod v1.Pod, replicaSetRef *metav1.OwnerReference) (restartAction, error) {

	replicaSetKey := client.ObjectKey{
		Namespace: pod.Namespace,
		Name:      replicaSetRef.Name,
	}

	var replicaSet = &appsv1.ReplicaSet{}
	err := retry.RetryOnError(retry.DefaultRetry, func() error {
		return c.Get(ctx, replicaSetKey, replicaSet)
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return newOwnerNotFoundAction(pod), nil
		}
		return restartAction{}, err
	}

	if rsOwnedBy, exists := getReplicaSetOwner(replicaSet); !exists {
		// If the ReplicaSet is not managed by a parent resource(e.g. deployment), we need to delete the pods in the ReplicaSet to force a restart.
		return newDeleteAction(actionObjectFromPod(pod)), nil
	} else {
		return newRolloutAction(actionObject{
			Name:      rsOwnedBy.Name,
			Namespace: replicaSet.Namespace,
			Kind:      rsOwnedBy.Kind,
		}), nil
	}
}

// getOwnerReferences returns the owner reference of the pod and a boolean to verify if the owner reference exists or not
func getReplicaSetOwner(rs *appsv1.ReplicaSet) (*metav1.OwnerReference, bool) {
	if len(rs.OwnerReferences) == 0 {
		return &metav1.OwnerReference{}, false
	}

	return rs.OwnerReferences[0].DeepCopy(), true
}
