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

var checkForMemoryUsage = func(ctx context.Context, sc *godog.Scenario, _ error) (context.Context, error) {
	var memoryUsageLimit *resource.Quantity = resource.NewQuantity(128*1024*1024, resource.BinarySI)
	c, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	podList := &v1c.PodList{}
	err = c.List(ctx, podList, client.MatchingLabels{"app.kubernetes.io/component": "istio-operator.kyma-project.io"})
	if err != nil {
		return ctx, nil
	}
	if len(podList.Items) < 1 {
		return ctx, errors.New("Controller not found")
	}
	cpod := podList.Items[0]

	mc, err := metrics.NewForConfig(config.GetConfigOrDie())
	if err != nil {
		return ctx, err
	}
	pm, err := mc.MetricsV1beta1().PodMetricses(cpod.Namespace).Get(ctx, cpod.Name, metav1.GetOptions{})
	if err != nil {
		return ctx, err
	}

	mu := pm.Containers[0].Usage.Memory()
	fmt.Printf("POD NAME############ %s ###########", cpod.Name)
	fmt.Printf("MEM USAGE############# %s ###########", mu.String())
	if mu.Cmp(*memoryUsageLimit) == -1 {
		return ctx, errors.New("Memory usage over 128MB")
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
