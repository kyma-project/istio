package integration

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

var testAppTearDown = func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	if testApp, ok := getTestAppFromContext(ctx); ok {
		err := retry.Do(func() error {
			return removeObjectFromCluster(ctx, testApp)
		}, retryOpts...)

		return ctx, err
	}
	return ctx, nil
}

var istioCrTearDown = func(ctx context.Context, sc *godog.Scenario, err error) (context.Context, error) {
	if istio, ok := getIstioCrFromContext(ctx); ok {
		err := retry.Do(func() error {
			return removeObjectFromCluster(ctx, istio)
		}, retryOpts...)
		// TODO: This is added to workaround that Istio deletion needs some time to remove all resources. If we don't wait, we might
		//  try to install a new Istio version while the old version is still uninstalling.
		time.Sleep(10 * time.Second)
		return ctx, err
	}
	return ctx, nil
}

func removeObjectFromCluster(ctx context.Context, object client.Object) error {
	log.Println(fmt.Sprintf("Teardown %s", object.GetName()))

	k8sClient, err := getK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	deletePolicy := metav1.DeletePropagationForeground
	err = k8sClient.Delete(ctx, object, &client.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}
	log.Println(fmt.Sprintf("Deleted %s", object.GetName()))

	return nil
}
