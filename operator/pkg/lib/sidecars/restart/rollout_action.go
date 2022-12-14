package restart

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const rolloutTimeoutMessage = "pod could not be rolled out by resource owner's controller."

const annotationName = "istio-operator.kyma-project.io/restartedAt"

func newRolloutAction(object actionObject) restartAction {
	return restartAction{
		run:    rolloutRun,
		object: object,
	}
}

func rolloutRun(ctx context.Context, k8sclient client.Client, object actionObject) ([]RestartWarning, error) {
	var obj client.Object
	var err error

	switch object.Kind {
	case "DaemonSet":
		obj = &appsv1.DaemonSet{}
	case "Deployment":
		obj = &appsv1.Deployment{}
	case "ReplicaSet":
		obj = &appsv1.ReplicaSet{}
	case "StatefulSet":
		obj = &appsv1.StatefulSet{}
	default:
		return nil, fmt.Errorf("kind %s is not supported for rollout", object.Kind)
	}

	err = retry.RetryOnError(retry.DefaultBackoff, func() error {
		err := k8sclient.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: object.Namespace}, obj)
		if err != nil {
			return err
		}

		annotations := obj.GetAnnotations()
		if len(annotations) == 0 {
			annotations = map[string]string{}
		}

		annotations[annotationName] = time.Now().Format(time.RFC3339)
		obj.SetAnnotations(annotations)

		patch := client.StrategicMergeFrom(obj)
		return k8sclient.Patch(ctx, obj, patch)
	})

	if err != nil {
		return nil, err
	}

	return nil, nil
}
