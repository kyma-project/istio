package steps

import (
	"context"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/testcontext"
	apinetworkingv1 "istio.io/api/networking/v1"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateDestinationRule(ctx context.Context, name, namespace, host string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	d := networkingv1.DestinationRule{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "networking.istio.io/v1",
			Kind:       "DestinationRule",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apinetworkingv1.DestinationRule{
			Host: host,
		},
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &d)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, &d)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}
