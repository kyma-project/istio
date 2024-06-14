package steps

import (
	"context"
	"github.com/avast/retry-go"
	testcontext2 "github.com/kyma-project/istio/operator/tests/testcontext"
	networkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateDestinationRule(ctx context.Context, name, namespace, host string) (context.Context, error) {
	k8sClient, err := testcontext2.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	d := v1alpha3.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1beta1",
			Kind:       "DestinationRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: networkingv1alpha3.DestinationRule{
			Host: host,
		},
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &d)
		if err != nil {
			return err
		}
		ctx = testcontext2.AddCreatedTestObjectInContext(ctx, &d)
		return nil
	}, testcontext2.GetRetryOpts()...)

	return ctx, err
}
