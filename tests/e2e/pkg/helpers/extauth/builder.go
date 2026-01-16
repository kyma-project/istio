package extauth

import (
	"bytes"
	_ "embed"
	"fmt"
	"testing"
	"text/template"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

//go:embed manifest_template.yaml
var manifestTemplate string

// DeploymentInfo contains information about a deployed external authorizer instance
type DeploymentInfo struct {
	// Name is the name of the external authorizer service
	Name string
	// Namespace is the namespace where external authorizer is deployed
	Namespace string
	// HttpPort is the HTTP service port
	HttpPort int
	// GrpcPort is the gRPC service port
	GrpcPort int
	// Host is the hostname to use for requests (servicename.namespace.svc.cluster.local)
	Host string
}

// Builder provides a fluent API for creating external authorizer resources
type Builder struct {
	name      string
	namespace string
}

// NewBuilder creates a new external authorizer builder with default values
func NewBuilder() *Builder {
	return &Builder{
		name:      "ext-authz",
		namespace: "ext-auth",
	}
}

// WithName sets the name for all external authorizer resources (Service, Deployment)
func (b *Builder) WithName(name string) *Builder {
	b.name = name
	return b
}

// WithNamespace sets the namespace where external authorizer will be deployed
func (b *Builder) WithNamespace(namespace string) *Builder {
	b.namespace = namespace
	return b
}

// DeployWithCleanup deploys the external authorizer instance with the configured settings and registers a cleanup function
func (b *Builder) DeployWithCleanup(t *testing.T) (*DeploymentInfo, error) {
	t.Helper()

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return nil, fmt.Errorf("failed to get resources client: %w", err)
	}

	// Check if external authorizer already exists
	err = r.Get(t.Context(), b.name, b.namespace, &unstructured.Unstructured{})
	if err == nil {
		t.Logf("External authorizer %s already exists in namespace %s, skipping creation", b.name, b.namespace)
		return &DeploymentInfo{
			Name:      b.name,
			Namespace: b.namespace,
			HttpPort:  8000,
			GrpcPort:  9000,
			Host:      fmt.Sprintf("%s.%s.svc.cluster.local", b.name, b.namespace),
		}, nil
	}

	// Create namespace if it doesn't exist
	err = infrahelpers.CreateNamespace(t, b.namespace,
		infrahelpers.WithLabels(map[string]string{"kubernetes.io/metadata.name": b.namespace}),
		infrahelpers.IgnoreAlreadyExists())
	if err != nil {
		t.Logf("Failed to create namespace for external authorizer: %v", err)
		return nil, fmt.Errorf("failed to create namespace: %w", err)
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
		HttpPort:  8000,
		GrpcPort:  9000,
		Host:      fmt.Sprintf("%s.%s.svc.cluster.local", b.name, b.namespace),
	}, nil
}

func (b *Builder) generateManifest() ([]byte, error) {
	tmpl, err := template.New("extauth").Parse(manifestTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	data := struct {
		Name string
	}{
		Name: b.name,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}

func (b *Builder) deployWithCleanup(t *testing.T, r *resources.Resources, manifest []byte) error {
	t.Logf("Creating external authorizer %s in namespace %s", b.name, b.namespace)
	t.Logf("Applying manifest:\n%s", string(manifest))

	err := decoder.DecodeEach(
		t.Context(),
		bytes.NewBuffer(manifest),
		decoder.CreateHandler(r),
		decoder.MutateNamespace(b.namespace),
	)
	if err != nil {
		t.Logf("Failed to deploy external authorizer: %v", err)
		return fmt.Errorf("failed to deploy external authorizer: %w", err)
	}

	setup.DeclareCleanup(t, func() {
		t.Logf("Cleaning up external authorizer %s in namespace %s", b.name, b.namespace)
		err := decoder.DecodeEach(
			setup.GetCleanupContext(),
			bytes.NewBuffer(manifest),
			decoder.DeleteHandler(r),
			decoder.MutateNamespace(b.namespace),
		)
		if err != nil {
			t.Logf("Failed to clean up external authorizer: %v", err)
		} else {
			t.Logf("Successfully cleaned up external authorizer %s in namespace %s", b.name, b.namespace)
		}
	})

	return wait.For(conditions.New(r).DeploymentAvailable(b.name, b.namespace))
}
