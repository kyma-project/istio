package httpbin

import (
	"bytes"
	_ "embed"
	"fmt"
	"testing"
	"text/template"

	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

//go:embed manifest_template.yaml
var manifestTemplate string

// DeploymentInfo contains information about a deployed httpbin instance
type DeploymentInfo struct {
	// Name is the name of the httpbin service
	Name string
	// Namespace is the namespace where httpbin is deployed
	Namespace string
	// Port is the service port
	Port int
	// Host is the hostname to use for requests (servicename.namespace.svc.cluster.local)
	Host string
}

// Builder provides a fluent API for creating httpbin resources
type Builder struct {
	name        string
	namespace   string
	annotations map[string]string
	labels      map[string]string
}

// NewBuilder creates a new httpbin builder with default name "httpbin"
func NewBuilder() *Builder {
	return &Builder{
		name:        "httpbin",
		annotations: make(map[string]string),
		labels:      make(map[string]string),
	}
}

// WithName sets the name for all httpbin resources (ServiceAccount, Service, Deployment)
func (b *Builder) WithName(name string) *Builder {
	b.name = name
	return b
}

// WithNamespace sets the namespace where httpbin will be deployed
func (b *Builder) WithNamespace(namespace string) *Builder {
	b.namespace = namespace
	return b
}

// WithAnnotation adds a single annotation to the pod template
func (b *Builder) WithAnnotation(key, value string) *Builder {
	b.annotations[key] = value
	return b
}

// WithAnnotations adds multiple annotations to the pod template
func (b *Builder) WithAnnotations(annotations map[string]string) *Builder {
	for k, v := range annotations {
		b.annotations[k] = v
	}
	return b
}

// WithLabel adds a single label to all resources
func (b *Builder) WithLabel(key, value string) *Builder {
	b.labels[key] = value
	return b
}

// WithLabels adds multiple labels to all resources
func (b *Builder) WithLabels(labels map[string]string) *Builder {
	for k, v := range labels {
		b.labels[k] = v
	}
	return b
}

// WithRegularSidecar adds the annotation to use regular (non-native) sidecar
func (b *Builder) WithRegularSidecar() *Builder {
	b.annotations["sidecar.istio.io/nativeSidecar"] = "false"
	return b
}

// DeployWithCleanup deploys the httpbin instance with the configured settings and registers a cleanup function
func (b *Builder) DeployWithCleanup(t *testing.T) (*DeploymentInfo, error) {
	t.Helper()

	if b.namespace == "" {
		b.namespace = "default"
	}

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	manifest, err := b.generateManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifest: %w", err)
	}

	err = b.deployWithCleanup(t, r, manifest)
	if err != nil {
		return nil, err
	}

	return &DeploymentInfo{
		Name:      b.name,
		Namespace: b.namespace,
		Port:      8000,
		Host:      fmt.Sprintf("%s.%s.svc.cluster.local", b.name, b.namespace),
	}, nil
}

func (b *Builder) generateManifest() ([]byte, error) {
	tmpl, err := template.New("httpbin").Parse(manifestTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		Name        string
		Annotations map[string]string
		Labels      map[string]string
	}{
		Name:        b.name,
		Annotations: b.annotations,
		Labels:      b.labels,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

func (b *Builder) deployWithCleanup(t *testing.T, r *resources.Resources, manifest []byte) error {
	err := decoder.DecodeEach(
		t.Context(),
		bytes.NewBuffer(manifest),
		decoder.CreateHandler(r),
		decoder.MutateNamespace(b.namespace),
	)
	if err != nil {
		t.Logf("Failed to deploy httpbin: %v", err)
		return err
	}

	setup.DeclareCleanup(t, func() {
		t.Logf("Cleaning up httpbin %s in namespace %s", b.name, b.namespace)
		err := decoder.DecodeEach(
			setup.GetCleanupContext(),
			bytes.NewBuffer(manifest),
			decoder.DeleteHandler(r),
			decoder.MutateNamespace(b.namespace),
		)
		if err != nil {
			t.Logf("Failed to clean up httpbin: %v", err)
		} else {
			t.Logf("Successfully cleaned up httpbin %s in namespace %s", b.name, b.namespace)
		}
	})

	return wait.For(conditions.New(r).DeploymentAvailable(b.name, b.namespace))
}
