package modules

import (
	"bytes"
	_ "embed"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

//go:embed operator_v1alpha2_istio.yaml
var istioTemplate []byte

type IstioCROptions struct {
	Template string
}

func CreateIstioCR(t *testing.T, options ...IstioCROptions) error {
	t.Helper()
	t.Log("Creating Istio custom resource")

	r := infrahelpers.ResourcesClient(t)

	_ = v1alpha2.AddToScheme(r.GetScheme())

	template := istioTemplate
	if len(options) > 0 && options[0].Template != "" {
		template = []byte(options[0].Template)
	}

	icr := &v1alpha2.Istio{}
	require.NoError(t,
		decoder.Decode(
			bytes.NewBuffer(template),
			icr,
		),

		r.Create(t.Context(), icr),
	)

	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up Istio after the tests")
		require.NoError(t, TeardownIstioCR(t, options...))
	})

	t.Log("Waiting for Istio custom resource to be ready")
	clock := time.Now()
	require.NoError(t, wait.For(conditions.New(r).ResourceMatch(icr, func(obj k8s.Object) bool {
		t.Logf("Waiting for Istio custom resource %s to be ready", obj.GetName())
		t.Logf("Elapsed time: %s", time.Since(clock))

		icrObj, ok := obj.(*v1alpha2.Istio)
		if !ok {
			return false
		}
		return icrObj.Status.State == v1alpha2.Ready
	})))

	t.Log("Istio custom resource created successfully")
	return nil
}

func TeardownIstioCR(t *testing.T, options ...IstioCROptions) error {
	t.Helper()
	t.Log("Beginning cleanup of Istio custom resource")

	r := infrahelpers.ResourcesClient(t)

	require.NoError(t, v1alpha2.AddToScheme(r.GetScheme()))

	template := istioTemplate
	if len(options) > 0 && options[0].Template != "" {
		template = []byte(options[0].Template)
	}

	icr := &v1alpha2.Istio{}
	t.Log("Deleting Istio custom resource")
	err := decoder.Decode(
		bytes.NewBuffer(template),
		icr,
	)
	assert.NoError(t, err)

	err = r.Delete(setup.GetCleanupContext(), icr)
	assert.NoError(t, err)

	err = wait.For(conditions.New(r).ResourceDeleted(icr), wait.WithTimeout(time.Minute*2))
	if err != nil {
		t.Logf("Failed to delete Istio custom resource: %v", err)
		_icr := &v1alpha2.Istio{}
		assert.NoError(t, r.Get(setup.GetCleanupContext(), icr.GetName(), icr.GetNamespace(), _icr))
		t.Logf("Istio custom resource still exists: state=%s, description=%s",
			_icr.Status.State, _icr.Status.Description)

		return err
	}

	t.Log("Istio custom resource deleted successfully")
	return nil
}
