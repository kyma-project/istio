package modules

import (
	"bytes"
	_ "embed"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

//go:embed operator_v1alpha2_istio.yaml
var istioTemplate []byte

type IstioCROptions struct {
	Template []byte
}

func WithIstioTemplate(template string) IstioCROption {
	return func(opts *IstioCROptions) {
		opts.Template = []byte(template)
	}
}

type IstioCROption func(*IstioCROptions)

func CreateIstioCR(t *testing.T, options ...IstioCROption) error {
	t.Helper()
	t.Log("Creating Istio custom resource")
	opts := &IstioCROptions{
		Template: istioTemplate,
	}
	for _, opt := range options {
		opt(opts)
	}

	r, err := infrahelpers.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	err = v1alpha2.AddToScheme(r.GetScheme())
	if err != nil {
		t.Logf("Failed to add Istio v1alpha2 scheme: %v", err)
		return err
	}

	icr := &v1alpha2.Istio{}
	err = decoder.Decode(
		bytes.NewBuffer(opts.Template),
		icr,
	)
	if err != nil {
		t.Logf("Failed to decode Istio custom resource template: %v", err)
		return err
	}

	err = r.Create(t.Context(), icr)
	if err != nil {
		t.Logf("Failed to create Istio custom resource: %v", err)
		return err
	}

	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up Istio after the tests")
		err := TeardownIstioCR(t, options...)
		if err != nil {
			t.Logf("Failed to clean up Istio custom resource: %v", err)
		} else {
			t.Log("Istio custom resource cleaned up successfully")
		}
	})

	err = waitForICRReadiness(t, r, icr)
	if err != nil {
		t.Logf("Failed to wait for Istio custom resource readiness: %v", err)
		return err
	}

	t.Log("Istio custom resource created successfully")
	return nil
}

func waitForICRReadiness(t *testing.T, r *resources.Resources, icr *v1alpha2.Istio) error {
	t.Helper()
	t.Log("Waiting for Istio custom resource to be ready")

	clock := time.Now()
	err := wait.For(conditions.New(r).ResourceMatch(icr, func(obj k8s.Object) bool {
		t.Logf("Waiting for Istio custom resource %s to be ready", obj.GetName())
		t.Logf("Elapsed time: %s", time.Since(clock))

		icrObj, ok := obj.(*v1alpha2.Istio)
		if !ok {
			return false
		}
		return icrObj.Status.State == v1alpha2.Ready
	}))
	if err != nil {
		t.Logf("Failed to wait for Istio custom resource to be ready: %v", err)
		return err
	}

	t.Log("Istio custom resource is ready")
	return nil
}

const icrDeletionTimeout = time.Minute * 2

func waitForICRDeletion(t *testing.T, r *resources.Resources, icr *v1alpha2.Istio) error {
	t.Helper()
	t.Log("Waiting for Istio custom resource to be deleted")

	err := wait.For(conditions.New(r).ResourceDeleted(icr), wait.WithTimeout(icrDeletionTimeout))
	if err != nil {
		t.Logf("Failed to wait for Istio custom resource deletion: %v", err)
		return err
	}

	t.Log("Istio custom resource deleted successfully")
	return nil
}

func TeardownIstioCR(t *testing.T, options ...IstioCROption) error {
	t.Helper()
	t.Log("Beginning cleanup of Istio custom resource")
	opts := &IstioCROptions{
		Template: istioTemplate,
	}
	for _, opt := range options {
		opt(opts)
	}

	r, err := infrahelpers.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	err = v1alpha2.AddToScheme(r.GetScheme())
	if err != nil {
		t.Logf("Failed to add Istio v1alpha2 scheme: %v", err)
		return err
	}

	icr := &v1alpha2.Istio{}
	t.Log("Deleting Istio custom resource")
	err = decoder.Decode(
		bytes.NewBuffer(opts.Template),
		icr,
	)
	if err != nil {
		t.Logf("Failed to decode Istio custom resource template: %v", err)
		return err
	}

	err = r.Delete(setup.GetCleanupContext(), icr)
	if err != nil {
		t.Logf("Failed to delete Istio custom resource: %v", err)
		if k8serrors.IsNotFound(err) {
			t.Log("Istio custom resource not found, nothing to delete")
			return nil
		}
		return err
	}

	return waitForICRDeletion(t, r, icr)
}
