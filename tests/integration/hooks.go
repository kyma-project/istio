package integration

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/pkg/errors"
	v1c "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metrics "k8s.io/metrics/pkg/client/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var testObjectsTearDown = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
	if objects, ok := testcontext.GetCreatedTestObjectsFromContext(ctx); ok {
		for _, o := range objects {
			err := retry.Do(func() error {
				return removeObjectFromCluster(ctx, o)
			}, testcontext.GetRetryOpts()...)

			if err != nil {
				return ctx, err
			}
		}
	}
	return ctx, nil
}

var istioCrTearDown = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {

	if istio, ok := testcontext.GetIstioCrFromContext(ctx); ok {
		// We can ignore a failed removal of the Istio CR, because we need to run force remove in any case to make sure no resource is left before the next scenario
		_ = retry.Do(func() error {
			return removeObjectFromCluster(ctx, istio)
		}, testcontext.GetRetryOpts()...)
		err := forceIstioCrRemoval(ctx, istio)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

var verifyControllerMemoryUsage = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
	const memoryLimitInMb = 128 // Memory usage limit for controller above which tests will fail (in MB)
	ml := resource.NewScaledQuantity(memoryLimitInMb, resource.Mega)

	c, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	mc, err := metrics.NewForConfig(config.GetConfigOrDie())
	if err != nil {
		return ctx, err
	}

	podList := &v1c.PodList{}
	err = c.List(ctx, podList, client.MatchingLabels{"app.kubernetes.io/component": "istio-operator.kyma-project.io"})
	if err != nil {
		return ctx, err
	}
	if len(podList.Items) < 1 {
		return ctx, errors.New("Controller not found")
	}

	for _, cpod := range podList.Items {
		pm, err := mc.MetricsV1beta1().PodMetricses(cpod.Namespace).Get(ctx, cpod.Name, metav1.GetOptions{})
		if err != nil {
			return ctx, err
		}

		mu := pm.Containers[0].Usage.Memory()
		if mu.Cmp(*ml) == 1 {
			//conv from B to MB
			limMB := ml.AsApproximateFloat64() / 1024 / 1024
			currMB := mu.AsApproximateFloat64() / 1024 / 1024

			errMsg := fmt.Sprintf("Controller memory usage over %.1fMB (%.1fMB) ", limMB, currMB)
			return ctx, errors.New(errMsg)
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

		return errors.New("Istio CR found and not in error state, force removal not necessary yet")
	}, testcontext.GetRetryOpts()...)
}

func removeObjectFromCluster(ctx context.Context, object client.Object) error {
	t, err := testcontext.GetTestingFromContext(ctx)
	if err != nil {
		return err
	}

	t.Logf("Teardown %s", object.GetName())

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
	t.Logf("Deleted %s", object.GetName())

	return nil
}
