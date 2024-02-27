package integration

import (
	"context"
	"fmt"

	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/pkg/errors"
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

var istioCrTearDown = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {

	t, err := testcontext.GetTestingFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	if istios, ok := testcontext.GetIstioCRsFromContext(ctx); ok {
		// We can ignore a failed removal of the Istio CR, because we need to run force remove in any case to make sure no resource is left before the next scenario
		for _, istio := range istios {
			_ = retry.Do(func() error {
				err := removeObjectFromCluster(ctx, istio)
				if err != nil {
					t.Logf("Failed to deleted Istio CR %s", istio.GetName())
					return err
				}
				t.Logf("Deleted Istio CR %s", istio.GetName())
				return nil
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

func forceIstioCrRemoval(ctx context.Context, istio *v1alpha2.Istio) error {
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

		if istio.Status.State == v1alpha2.Error {
			t.Logf("Istio CR in error state (%s), force removal", istio.Status.Description)
			istio.Finalizers = nil
			err = c.Update(ctx, istio)
			if err != nil {
				return err
			}

			return nil
		}

		return errors.New(fmt.Sprintf("istio CR in status %s found (%s), skipping force removal", istio.Status.State, istio.Status.Description))
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
