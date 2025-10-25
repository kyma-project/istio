package modules

import (
	"bytes"
	_ "embed"
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"sigs.k8s.io/e2e-framework/klient/k8s"
	"testing"
	"text/template"
	"time"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

//go:embed operator_v1alpha2_istio_default.yaml
var IstioDefaultTemplate string

type IstioCROptions struct {
	Template       []byte
	TemplateValues map[string]interface{}
}

type IstioCROption func(options *IstioCROptions)

func WithIstioOperatorTemplate(template string) IstioCROption {
	return func(opts *IstioCROptions) {
		opts.Template = []byte(template)
	}
}

func WithIstioOperatorTemplateValues(values map[string]interface{}) IstioCROption {
	return func(opts *IstioCROptions) {
		opts.TemplateValues = values
	}
}

func CreateIstioOperatorCR(t *testing.T, options ...IstioCROption) error {
	t.Helper()
	t.Log("Creating Istio custom resource")
	opts := &IstioCROptions{
		Template:       []byte(IstioDefaultTemplate),
		TemplateValues: map[string]interface{}{},
	}
	for _, opt := range options {
		opt(opts)
	}

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	icr := &v1alpha2.Istio{}

	tmpl, err := template.New("").Option("missingkey=error").Parse(string(opts.Template))
	if err != nil {
		t.Logf("Failed to parse resource template %s: %v", opts.Template, err)
		return err
	}
	var tmplBuffer bytes.Buffer
	err = tmpl.Execute(&tmplBuffer, opts.TemplateValues)
	if err != nil {
		t.Logf("Failed to execute template for resource %s with values %v: %v", opts.Template, opts.TemplateValues, err)
		return err
	}

	err = decoder.Decode(
		bytes.NewBuffer(tmplBuffer.Bytes()),
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

	err = waitForIstioCRReadiness(t, r, icr)
	if err != nil {
		t.Logf("Istio custom resource is not ready: %v", err)
		return err
	}

	t.Log("Istio custom resource created successfully")
	return nil
}

func TeardownIstioCR(t *testing.T, options ...IstioCROption) error {
	t.Helper()
	t.Log("Beginning cleanup of Istio custom resource")
	opts := &IstioCROptions{
		Template: []byte(IstioDefaultTemplate),
	}
	for _, opt := range options {
		opt(opts)
	}

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	icr := &unstructured.Unstructured{}
	tmpl, err := template.New("").Option("missingkey=error").Parse(string(opts.Template))
	if err != nil {
		t.Logf("Failed to parse resource template %s: %v", opts.Template, err)
		return err
	}
	var tmplBuffer bytes.Buffer
	err = tmpl.Execute(&tmplBuffer, opts.TemplateValues)
	if err != nil {
		t.Logf("Failed to execute template for resource %s with values %v: %v", opts.Template, opts.TemplateValues, err)
		return err
	}

	err = decoder.Decode(
		bytes.NewBuffer(tmplBuffer.Bytes()),
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

	return waitForIstioCRDeletion(t, r, icr)
}

var istioCRDeletionTimeout = 2 * time.Minute

func waitForIstioCRReadiness(t *testing.T, r *resources.Resources, istio *v1alpha2.Istio) error {
	t.Helper()
	t.Log("Waiting for Istio custom resource to be ready")

	clock := time.Now()

	err := wait.For(conditions.New(r).ResourceMatch(istio, func(obj k8s.Object) bool {
		istioCR := obj.(*v1alpha2.Istio)

		t.Logf("Waiting for Istio custom resource to be ready; name: %s, namespace: %s", obj.GetName(), obj.GetNamespace())
		t.Logf("Elapsed time: %s", time.Since(clock))

		return istioCR.Status.State == v1alpha2.Ready
	}))

	if err != nil {
		t.Logf("Failed to wait for Istio custom resource to be ready: %v", err)
		if err != nil {
			t.Logf("Failed to get Istio custom resource: %v", err)
		} else {
			t.Logf("Istio custom resource status: %+v", istio.Status)
		}
		return err
	}

	t.Log("Istio custom resource is ready")
	return nil
}

func waitForIstioCRDeletion(t *testing.T, r *resources.Resources, istioCR *unstructured.Unstructured) error {
	t.Helper()
	t.Log("Waiting for Istio custom resource to be deleted")

	err := wait.For(conditions.New(r).ResourceDeleted(istioCR), wait.WithTimeout(istioCRDeletionTimeout))
	if err != nil {
		t.Logf("Failed to wait for Istio custom resource deletion: %v", err)
		return err
	}

	t.Log("Istio custom resource deleted successfully")
	return nil
}
