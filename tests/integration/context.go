package integration

import (
	"context"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	v1 "k8s.io/api/apps/v1"
)

// istioCrCtxKey is the key used to store the IstioCR used by a scenario in the context.Context.
type istioCrCtxKey struct{}

func getIstioCrFromContext(ctx context.Context) (*v1alpha1.Istio, bool) {
	istio, ok := ctx.Value(istioCrCtxKey{}).(*v1alpha1.Istio)
	return istio, ok
}

func setIstioCrInContext(ctx context.Context, istio *v1alpha1.Istio) context.Context {
	return context.WithValue(ctx, istioCrCtxKey{}, istio)
}

// testAppCtxKey is the key used to store the test app used by a scenario in the context.Context.
type testAppCtxKey struct{}

func getTestAppFromContext(ctx context.Context) (*v1.Deployment, bool) {
	istio, ok := ctx.Value(testAppCtxKey{}).(*v1.Deployment)
	return istio, ok
}

func setTestAppInContext(ctx context.Context, istio *v1.Deployment) context.Context {
	return context.WithValue(ctx, testAppCtxKey{}, istio)
}
