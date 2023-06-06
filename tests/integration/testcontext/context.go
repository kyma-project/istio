package testcontext

import (
	"context"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// istioCrCtxKey is the key used to store the IstioCR used by a scenario in the context.Context.
type istioCrCtxKey struct{}

func GetIstioCrFromContext(ctx context.Context) (*v1alpha1.Istio, bool) {
	v, ok := ctx.Value(istioCrCtxKey{}).(*v1alpha1.Istio)
	return v, ok
}

func SetIstioCrInContext(ctx context.Context, istio *v1alpha1.Istio) context.Context {
	return context.WithValue(ctx, istioCrCtxKey{}, istio)
}

// testAppCtxKey is the key used to store the test app used by a scenario in the context.Context.
type testAppCtxKey struct{}

func GetTestAppFromContext(ctx context.Context) (*v1.Deployment, bool) {
	v, ok := ctx.Value(testAppCtxKey{}).(*v1.Deployment)
	return v, ok
}

func SetTestAppInContext(ctx context.Context, istio *v1.Deployment) context.Context {
	return context.WithValue(ctx, testAppCtxKey{}, istio)
}

// k8sClientCtxKey is the key used to store the k8sClient used during tests in the context.Context.
type k8sClientCtxKey struct{}

func GetK8sClientFromContext(ctx context.Context) (client.Client, error) {
	v, ok := ctx.Value(k8sClientCtxKey{}).(client.Client)
	if !ok {
		return v, errors.New("k8sClient not found in context")
	}
	return v, nil
}

func SetK8sClientInContext(ctx context.Context, k8sClient client.Client) context.Context {
	return context.WithValue(ctx, k8sClientCtxKey{}, k8sClient)
}
