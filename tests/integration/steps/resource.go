package steps

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type godogResourceMapping int

func (k godogResourceMapping) String() string {
	switch k {
	case DaemonSet:
		return "DaemonSet"
	case Deployment:
		return "Deployment"
	case IstioCR:
		return "Istio CR"
	}
	panic(fmt.Errorf("%#v has unimplemented String() method", k))
}

const (
	DaemonSet godogResourceMapping = iota
	Deployment
	IstioCR
)

func ResourceIsReady(ctx context.Context, kind, name, namespace string) error {

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
		var object client.Object
		switch kind {
		case Deployment.String():
			object = &v1.Deployment{}
		case DaemonSet.String():
			object = &v1.DaemonSet{}
		default:
			return godog.ErrUndefined
		}
		err := k8sClient.Get(context.TODO(), client.ObjectKey{Namespace: namespace, Name: name}, object)
		if err != nil {
			return err
		}

		if err != nil {
			return err
		}

		switch kind {
		case Deployment.String():
			if object.(*v1.Deployment).Status.Replicas != object.(*v1.Deployment).Status.ReadyReplicas {
				return fmt.Errorf("%s %s/%s is not ready",
					kind, namespace, name)
			}
		case DaemonSet.String():
			if object.(*v1.DaemonSet).Status.NumberReady != object.(*v1.DaemonSet).Status.DesiredNumberScheduled {
				return fmt.Errorf("%s %s/%s is not ready",
					kind, namespace, name)
			}
		default:
			return godog.ErrUndefined
		}

		return nil
	}, testcontext.GetRetryOpts()...)
}

func NamespaceIsPresent(ctx context.Context, name, shouldBePresent string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	var ns corev1.Namespace
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
		if shouldBePresent != "present" {
			if !k8serrors.IsNotFound(err) {
				return fmt.Errorf("namespace %s is present but shouldn't", name)
			}
			return nil
		}
		return err
	}, testcontext.GetRetryOpts()...)
}

func NamespaceHasLabelAndAnnotation(ctx context.Context, name, label, annotation string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	var ns corev1.Namespace
	return retry.Do(func() error {
		err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name}, &ns)
		if _, ok := ns.Labels[label]; !ok {
			return fmt.Errorf("namespace %s does not contain %s label", name, label)
		}
		if _, ok := ns.Annotations[annotation]; !ok {
			return fmt.Errorf("namespace %s does not contain %s annotation", name, annotation)
		}
		return err
	}, testcontext.GetRetryOpts()...)
}

func ResourceInNamespaceIsDeleted(ctx context.Context, kind, name, namespace string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	switch kind {
	case IstioCR.String():
		return retry.Do(func() error {
			var istioCr istioCR.Istio
			err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, &istioCr)
			if err != nil {
				return err
			}

			return k8sClient.Delete(context.TODO(), &istioCr)
		})
	default:
		return godog.ErrUndefined
	}
}

func ResourceNotPresent(ctx context.Context, kind string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
		switch kind {
		case IstioCR.String():
			var istioList istioCR.IstioList
			err := k8sClient.List(context.TODO(), &istioList)
			if err != nil {
				return err
			}
			if len(istioList.Items) > 0 {
				return fmt.Errorf("there are %d %s present but shouldn't", len(istioList.Items), kind)
			}
		}
		return nil
	}, testcontext.GetRetryOpts()...)
}
