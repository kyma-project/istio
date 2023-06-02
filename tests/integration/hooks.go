package integration

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var testAppTearDown = func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	if testApp, ok := testcontext.GetTestAppFromContext(ctx); ok {
		err := retry.Do(func() error {
			return removeObjectFromCluster(ctx, testApp)
		}, testcontext.GetRetryOpts()...)

		return ctx, err
	}
	return ctx, nil
}

var istioCrTearDown = func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	// In case the scenario is an uninstall scenario, the istio CR deletion should be part of the scenario
	if testcontext.HasUninstallTag(sc) {
		return ctx, nil
	}

	if istio, ok := testcontext.GetIstioCrFromContext(ctx); ok {
		// We can ignore a failed removal of the Istio CR, because we need to run force remove in any case to make sure no resource is left before the next scenario
		_ = retry.Do(func() error {
			return removeObjectFromCluster(ctx, istio)
		}, testcontext.GetRetryOpts()...)
		err = forceIstioCrRemoval(ctx, istio)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

func forceIstioCrRemoval(ctx context.Context, istio *v1alpha1.Istio) error {
	return retry.Do(func() error {
		c, err := testcontext.GetK8sClientFromContext(ctx)
		if err != nil {
			return err
		}
		err = c.Get(ctx, client.ObjectKey{Namespace: istio.GetNamespace(), Name: istio.GetName()}, istio)

		if k8serrors.IsNotFound(err) {
			return nil
		}

		if err != nil {
			return err
		}

		if istio.Status.State == v1alpha1.Error {
			log.Println("Istio CR in error state, force removal")
			istio.Finalizers = nil
			err = c.Update(ctx, istio)
			if err != nil {
				return err
			}

			return nil
		}

		return errors.New("Istio CR found and not in error state, force removal not necessary yet")
	}, testcontext.GetRetryOpts()...)
}

func removeObjectFromCluster(ctx context.Context, object client.Object) error {
	log.Println(fmt.Sprintf("Teardown %s", object.GetName()))

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
	log.Println(fmt.Sprintf("Deleted %s", object.GetName()))

	return nil
}
