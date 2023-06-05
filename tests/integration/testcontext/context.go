package testcontext

import (
	"context"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/pkg/errors"
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

// createdTestObjectsCtxKey is the key used to store the test resources created during tests in the context.Context.
type createdTestObjectsCtxKey struct{}

func GetCreatedTestObjectsFromContext(ctx context.Context) ([]client.Object, bool) {
	v, ok := ctx.Value(createdTestObjectsCtxKey{}).([]client.Object)
	return v, ok
}

func AddCreatedTestObjectInContext(ctx context.Context, object client.Object) context.Context {
	objects, ok := GetCreatedTestObjectsFromContext(ctx)
	if !ok {
		objects = []client.Object{}
	}

	objects = append(objects, object)
	return context.WithValue(ctx, createdTestObjectsCtxKey{}, objects)
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
