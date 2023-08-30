package integration

import (
	"context"
	"fmt"

	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/pkg/errors"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testObjectsTearDown = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
	if objects, ok := testcontext.GetCreatedTestObjectsFromContext(ctx); ok {
		for _, o := range objects {

			t, err := testcontext.GetTestingFromContext(ctx)
			if err != nil {
				return ctx, err
			}
			t.Logf("Teardown %s", o.GetName())

			err = retry.Do(func() error {
				return removeObjectFromCluster(ctx, o)
			}, testcontext.GetRetryOpts()...)

			if err != nil {
				t.Logf("Failed to delete %s", o.GetName())
				return ctx, err
			}

			t.Logf("Deleted %s", o.GetName())
		}
	}
	return ctx, nil
}

var additionalResourcesTearDown = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
	t, err := testcontext.GetTestingFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	objects := []client.Object{}

	objects = append(objects, &networkingv1alpha3.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyma-gateway",
			Namespace: "kyma-system",
		},
	})
	objects = append(objects, &networkingv1alpha3.EnvoyFilter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kyma-referer",
			Namespace: "istio-system",
		},
	})
	objects = append(objects, &securityv1beta1.PeerAuthentication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "istio-system",
		},
	})
	objects = append(objects, &networkingv1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-healthz",
			Namespace: "istio-system",
		},
	})
	objects = append(objects, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-control-plane-grafana-dashboard",
			Namespace: "kyma-system",
		},
	})
	objects = append(objects, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-mesh-grafana-dashboard",
			Namespace: "kyma-system",
		},
	})
	objects = append(objects, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-performance-grafana-dashboard",
			Namespace: "kyma-system",
		},
	})
	objects = append(objects, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-service-grafana-dashboard",
			Namespace: "kyma-system",
		},
	})
	objects = append(objects, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "istio-workload-grafana-dashboard",
			Namespace: "kyma-system",
		},
	})

	for _, o := range objects {
		t.Logf("Teardown %s", o.GetName())

		err = retry.Do(func() error {
			return removeObjectFromCluster(ctx, o)
		}, testcontext.GetRetryOpts()...)

		if err != nil {
			t.Logf("Failed to delete %s", o.GetName())
			return ctx, err
		}

		t.Logf("Deleted %s", o.GetName())
	}

	return ctx, nil
}

var istioCrTearDown = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
	if istios, ok := testcontext.GetIstioCRsFromContext(ctx); ok {
		// We can ignore a failed removal of the Istio CR, because we need to run force remove in any case to make sure no resource is left before the next scenario
		for _, istio := range istios {
			_ = retry.Do(func() error {
				return removeObjectFromCluster(ctx, istio)
			}, testcontext.GetRetryOpts()...)
			err := forceIstioCrRemoval(ctx, istio)
			if err != nil {
				return ctx, err
			}
		}
	}
	return ctx, nil
}

var verifyIfControllerHasBeenRestarted = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
	c, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	podList := &corev1.PodList{}
	err = c.List(ctx, podList, client.MatchingLabels{"app.kubernetes.io/component": "istio-operator.kyma-project.io"})
	if err != nil {
		return ctx, err
	}
	if len(podList.Items) < 1 {
		return ctx, errors.New("Controller not found")
	}

	for _, cpod := range podList.Items {
		if len(cpod.Status.ContainerStatuses) > 0 {
			if rc := cpod.Status.ContainerStatuses[0].RestartCount; rc > 0 {
				errMsg := fmt.Sprintf("Controller has been restarted %d times", rc)
				return ctx, errors.New(errMsg)
			}
		}
	}

	return ctx, nil
}

func forceIstioCrRemoval(ctx context.Context, istio *v1alpha1.Istio) error {
	c, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	t, err := testcontext.GetTestingFromContext(ctx)
	if err != nil {
		return err
	}

	return retry.Do(func() error {

		err = c.Get(ctx, client.ObjectKey{Namespace: istio.GetNamespace(), Name: istio.GetName()}, istio)

		if k8serrors.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return err
		}

		if istio.Status.State == v1alpha1.Error {
			t.Log("Istio CR in error state, force removal")
			istio.Finalizers = nil
			err = c.Update(ctx, istio)
			if err != nil {
				return err
			}

			return nil
		}

		return errors.New(fmt.Sprintf("istio CR in status %s found, skipping force removal", istio.Status.State))
	}, testcontext.GetRetryOpts()...)
}

func removeObjectFromCluster(ctx context.Context, object client.Object) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	deletePolicy := metav1.DeletePropagationForeground
	err = k8sClient.Delete(context.TODO(), object, &client.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	return nil
}
