package modules

import (
	"bytes"
	"context"
	_ "embed"
	"testing"
	"text/template"
	"time"

	"sigs.k8s.io/e2e-framework/klient/k8s"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	appsv1 "k8s.io/api/apps/v1"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
		err := teardownIstioCR(t, icr)
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

func UpdateIstioOperatorCR(t *testing.T, options ...IstioCROption) error {
	t.Helper()
	t.Log("Updating Istio custom resource")
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
		err := teardownIstioCR(t, icr)
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

func teardownIstioCR(t *testing.T, istioCR *v1alpha2.Istio) error {
	t.Helper()
	t.Log("Beginning cleanup of Istio custom resource")
	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	err = r.Delete(setup.GetCleanupContext(), istioCR)
	if err != nil {
		t.Logf("Failed to delete Istio custom resource: %v", err)
		if k8serrors.IsNotFound(err) {
			t.Log("Istio custom resource not found, nothing to delete")
			return nil
		}
		return err
	}

	return waitForIstioCRDeletion(t, r, istioCR)
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

// waitForIstioCRReadinessWithContext is like waitForIstioCRReadiness but accepts an explicit context.
// Used when t.Context() is no longer valid — for example inside t.Cleanup, where t.Context()
// is already cancelled and would cause the wait to exit immediately.
func waitForIstioCRReadinessWithContext(ctx context.Context, t *testing.T, r *resources.Resources, istio *v1alpha2.Istio) error {
	t.Helper()
	t.Log("Waiting for Istio custom resource to be ready")

	clock := time.Now()

	err := wait.For(conditions.New(r).ResourceMatch(istio, func(obj k8s.Object) bool {
		istioCR := obj.(*v1alpha2.Istio)

		t.Logf("Waiting for Istio custom resource to be ready; name: %s, namespace: %s", obj.GetName(), obj.GetNamespace())
		t.Logf("Elapsed time: %s", time.Since(clock))

		return istioCR.Status.State == v1alpha2.Ready
	}), wait.WithContext(ctx))

	if err != nil {
		t.Logf("Failed to wait for Istio custom resource to be ready: %v", err)
		t.Logf("Istio custom resource status: %+v", istio.Status)
		return err
	}

	t.Log("Istio custom resource is ready")
	return nil
}

func waitForIstioCRDeletion(t *testing.T, r *resources.Resources, istioCR *v1alpha2.Istio) error {
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

// WaitForIngressGatewayReplicas waits until the istio-ingressgateway Deployment reaches the expected replica count.
func WaitForIngressGatewayReplicas(ctx context.Context, t *testing.T, r *resources.Resources, expected int32) error {
	t.Helper()
	return wait.For(func(ctx context.Context) (bool, error) {
		dep := &appsv1.Deployment{}
		if err := r.Get(ctx, "istio-ingressgateway", "istio-system", dep); err != nil {
			return false, err
		}
		ready := dep.Status.Replicas == expected &&
			dep.Status.ReadyReplicas == expected &&
			dep.Status.UpdatedReplicas == expected &&
			dep.Status.AvailableReplicas == expected &&
			dep.Status.ObservedGeneration >= dep.Generation
		if !ready {
			t.Logf("waiting for ingress gateway: total=%d ready=%d updated=%d available=%d (want %d)",
				dep.Status.Replicas, dep.Status.ReadyReplicas, dep.Status.UpdatedReplicas, dep.Status.AvailableReplicas, expected)
		}
		return ready, nil
	}, wait.WithTimeout(5*time.Minute), wait.WithContext(ctx))
}
