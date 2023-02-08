package restart

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const restartAnnotationName = "istio-operator.kyma-project.io/restartedAt"

func newRolloutAction(object actionObject) restartAction {
	return restartAction{
		run:    rolloutRun,
		object: object,
	}
}

func rolloutRun(ctx context.Context, k8sclient client.Client, object actionObject, logger *logr.Logger) ([]RestartWarning, error) {
	logger.Info("Roll out pod due to proxy restart", "name", object.Name, "namespace", object.Namespace)

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
			ds.Spec.Template.Annotations = addRestartAnnotation(ds.Spec.Template.Annotations)

			return k8sclient.Update(ctx, ds)
		})
	case "Deployment":
		obj = &appsv1.Deployment{}
		err = retry.RetryOnError(retry.DefaultBackoff, func() error {
			err := k8sclient.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: object.Namespace}, obj)
			if err != nil {
				return err
			}
			dep := obj.(*appsv1.Deployment)
			dep.Spec.Template.Annotations = addRestartAnnotation(dep.Spec.Template.Annotations)

			return k8sclient.Update(ctx, dep)
		})
	case "ReplicaSet":
		obj = &appsv1.ReplicaSet{}
		err = retry.RetryOnError(retry.DefaultBackoff, func() error {
			err := k8sclient.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: object.Namespace}, obj)
			if err != nil {
				return err
			}
			rs := obj.(*appsv1.ReplicaSet)
			rs.Spec.Template.Annotations = addRestartAnnotation(rs.Spec.Template.Annotations)

			return k8sclient.Update(ctx, rs)
		})
	case "StatefulSet":
		obj = &appsv1.StatefulSet{}
		err = retry.RetryOnError(retry.DefaultBackoff, func() error {
			err := k8sclient.Get(ctx, types.NamespacedName{Name: object.Name, Namespace: object.Namespace}, obj)
			if err != nil {
				return err
			}
			ss := obj.(*appsv1.StatefulSet)
			ss.Spec.Template.Annotations = addRestartAnnotation(ss.Spec.Template.Annotations)

			return k8sclient.Update(ctx, ss)
		})
	default:
		return nil, fmt.Errorf("kind %s is not supported for rollout", object.Kind)
	}

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func addRestartAnnotation(annotations map[string]string) map[string]string {
	if len(annotations) == 0 {
		annotations = map[string]string{}
	}

	annotations[restartAnnotationName] = time.Now().Format(time.RFC3339)
	return annotations
}
