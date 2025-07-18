package oauth2mock

import (
	"bytes"
	_ "embed"
	"fmt"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"testing"
	"text/template"
)

//go:embed manifest.yaml
var rawManifest string

type Mock struct {
	IssuerURL, TokenURL string
	Subdomain           string

	parsedManifest []byte
}

type Options struct {
	Namespace string
}

func WithNamespace(ns string) Option {
	return func(o *Options) {
		o.Namespace = ns
	}
}

type Option func(*Options)

func DeployMock(t *testing.T, domain string, options ...Option) (*Mock, error) {
	t.Helper()
	opts := &Options{
		Namespace: "oauth2-mock",
	}
	for _, opt := range options {
		opt(opts)
	}

	mock := &Mock{
		IssuerURL: fmt.Sprintf("mock-oauth2-server.%s.svc.cluster.local", opts.Namespace),
		TokenURL:  fmt.Sprintf("https://%s.%s/oauth2/token", opts.Namespace, domain),
		Subdomain: fmt.Sprintf("%s.%s", opts.Namespace, domain),
	}

	t.Logf("Deploying oauth2mock with IssuerURL: %s, TokenURL: %s, Subdomain: %s",
		mock.IssuerURL, mock.TokenURL, mock.Subdomain)
	return mock, startMock(t, mock, opts)
}

func startMock(t *testing.T, m *Mock, options *Options) error {
	t.Helper()
	r, err := infrahelpers.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	err = infrahelpers.CreateNamespace(t, options.Namespace, infrahelpers.IgnoreAlreadyExists())
	if err != nil {
		t.Logf("Failed to create namespace: %v", err)
		return fmt.Errorf("failed to create namespace %s: %w", options.Namespace, err)
	}

	// No further cleanup is needed as the namespace will be deleted
	// as part of Namespace cleanup.
	// setup.DeclareCleanup(t, func() {})

	return m.start(t, r, options)
}

func (m *Mock) start(t *testing.T, r *resources.Resources, options *Options) error {
	err := m.parseTmpl()
	if err != nil {
		return err
	}

	err = decoder.DecodeEach(
		t.Context(),
		bytes.NewBuffer(m.parsedManifest),
		decoder.CreateHandler(r),
		decoder.MutateNamespace(options.Namespace),
	)
	if err != nil {
		t.Logf("Failed to deploy mock: %v", err)
		return err
	}

	return wait.For(conditions.New(r).DeploymentAvailable("mock-oauth2-server-deployment", options.Namespace))
}

func (m *Mock) parseTmpl() error {
	var sbuf bytes.Buffer
	tmpl, err := template.New("").Parse(rawManifest)
	if err != nil {
		return err
	}
	err = tmpl.Execute(&sbuf, m)
	if err != nil {
		return err
	}
	m.parsedManifest = sbuf.Bytes()
	return nil
}
