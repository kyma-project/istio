package helpers

import (
	"io/fs"
	"testing"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

func SetupIstio(t *testing.T, fsys fs.FS, cfg *envconf.Config) error {
	t.Helper()
	r, err := resources.New(WrapTestLog(t, cfg.Client().RESTConfig()))
	if err != nil {
		return err
	}
	_ = v1alpha2.AddToScheme(r.GetScheme())

	icr := &v1alpha2.Istio{}
	err = decoder.DecodeFile(fsys, "istio_customresource.yaml", icr)
	if err != nil {
		return err
	}
	err = r.Create(t.Context(), icr)
	if err != nil {
		return err
	}

	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up Istio after the tests")
		require.NoError(t, TeardownIstio(t, fsys, cfg))
	})
	// Wait for Istio to be ready
	err = wait.For(conditions.New(r).ResourceMatch(icr, func(obj k8s.Object) bool {
		icrObj, ok := obj.(*v1alpha2.Istio)
		if !ok {
			return false
		}
		return icrObj.Status.State == v1alpha2.Ready
	}))

	return nil
}
func TeardownIstio(t *testing.T, fsys fs.FS, cfg *envconf.Config) error {
	t.Helper()
	r, err := resources.New(WrapTestLog(t, cfg.Client().RESTConfig()))
	if err != nil {
		return err
	}
	_ = v1alpha2.AddToScheme(r.GetScheme())

	icr := &v1alpha2.Istio{}
	err = decoder.DecodeFile(fsys, "istio_customresource.yaml", icr, decoder.MutateNamespace("kyma-system"))
	if err != nil {
		return err
	}

	err = r.Delete(setup.GetCleanupContext(), icr)
	if err != nil {
		return err
	}

	err = wait.For(conditions.New(r).ResourceDeleted(icr))
	if err != nil {
		return err
	}

	return nil
}
