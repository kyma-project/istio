package restart

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"

	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newRolloutAction(object actionObject) restartAction {
	return restartAction{
		run:    rolloutRun,
		object: object,
	}
}

func rolloutRun(ctx context.Context, k8sclient client.Client, object actionObject, logger *logr.Logger) ([]RestartWarning, error) {
	logger.Info("Rollout pod due to proxy restart", "name", object.Name, "namespace", object.Namespace, "kind", object.Kind)

	var obj client.Object
	var err error

	switch object.Kind {
	case "DaemonSet":
		obj = &appsv1.DaemonSet{}
		err = retry.RetryOnError(retry.DefaultBackoff, func() error {
			err := k8sclient.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: object.Namespace}, obj)
			if err != nil {
				return err
			}
			ds := obj.(*appsv1.DaemonSet)
			patch := client.StrategicMergeFrom(ds.DeepCopy())
			ds.Spec.Template.Annotations = annotations.AddRestartAnnotation(ds.Spec.Template.Annotations)
			return k8sclient.Patch(ctx, ds, patch)
		})
	case "Deployment":
		obj = &appsv1.Deployment{}
		err = retry.RetryOnError(retry.DefaultBackoff, func() error {
			err := k8sclient.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: object.Namespace}, obj)
			if err != nil {
				return err
			}
			dep := obj.(*appsv1.Deployment)
			patch := client.StrategicMergeFrom(dep.DeepCopy())
			dep.Spec.Template.Annotations = annotations.AddRestartAnnotation(dep.Spec.Template.Annotations)
			return k8sclient.Patch(ctx, dep, patch)
		})
	case "ReplicaSet":
		obj = &appsv1.ReplicaSet{}
		err = retry.RetryOnError(retry.DefaultBackoff, func() error {
			err := k8sclient.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: object.Namespace}, obj)
			if err != nil {
				return err
			}
			rs := obj.(*appsv1.ReplicaSet)
			patch := client.StrategicMergeFrom(rs.DeepCopy())
			rs.Spec.Template.Annotations = annotations.AddRestartAnnotation(rs.Spec.Template.Annotations)
			return k8sclient.Patch(ctx, rs, patch)
		})
	case "StatefulSet":
		obj = &appsv1.StatefulSet{}
		err = retry.RetryOnError(retry.DefaultBackoff, func() error {
			err := k8sclient.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: object.Namespace}, obj)
			if err != nil {
				return err
			}
			ss := obj.(*appsv1.StatefulSet)
			patch := client.StrategicMergeFrom(ss.DeepCopy())
			ss.Spec.Template.Annotations = annotations.AddRestartAnnotation(ss.Spec.Template.Annotations)
			return k8sclient.Patch(ctx, ss, patch)
		})
	default:
		return nil, fmt.Errorf("kind %s is not supported for rollout", object.Kind)
	}

	if err != nil {
		return nil, err
	}

	return nil, nil
}
