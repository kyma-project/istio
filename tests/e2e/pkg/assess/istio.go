package assess

import (
	"context"
	"io/fs"
	"testing"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

func SetupIstioStep(fsys fs.FS) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		require.NoError(t, err)
		_ = v1alpha2.AddToScheme(r.GetScheme())

		icr := &v1alpha2.Istio{}
		require.NoError(t, decoder.DecodeFile(fsys, "istio_customresource.yaml", icr))
		require.NoError(t, r.Create(t.Context(), icr))

		// Wait for Istio to be ready
		require.NoError(t, wait.For(conditions.New(r).ResourceMatch(icr, func(obj k8s.Object) bool {
			icrObj, ok := obj.(*v1alpha2.Istio)
			if !ok {
				return false
			}
			return icrObj.Status.State == v1alpha2.Ready
		})))
		return ctx
	}
}

func TeardownIstioStep(fsys fs.FS) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		r, err := resources.New(cfg.Client().RESTConfig())
		require.NoError(t, err)
		_ = v1alpha2.AddToScheme(r.GetScheme())

		icr := &v1alpha2.Istio{}
		require.NoError(t, decoder.DecodeFile(fsys, "istio_customresource.yaml", icr, decoder.MutateNamespace("kyma-system")))
		require.NoError(t, r.Delete(ctx, icr))
		require.NoError(t, wait.For(conditions.New(r).ResourceDeleted(icr)))
		return ctx
	}
}
