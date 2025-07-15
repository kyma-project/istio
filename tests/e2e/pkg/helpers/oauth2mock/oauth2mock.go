package oauth2mock

import (
	"bytes"
	_ "embed"
	"fmt"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"github.com/stretchr/testify/require"
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

func DeployMock(t *testing.T, domain string, options ...Options) (*Mock, error) {
	t.Helper()

	mock := &Mock{
		IssuerURL: "mock-oauth2-server.oauth2-mock.svc.cluster.local",
		TokenURL:  fmt.Sprintf("https://oauth2-mock.%s/oauth2/token", domain),
		Subdomain: fmt.Sprintf("oauth2-mock.%s", domain),
	}
	if len(options) > 0 {
		if options[0].Namespace != "" {
			mock.IssuerURL = fmt.Sprintf("mock-oauth2-server.%s.svc.cluster.local", options[0].Namespace)
			mock.TokenURL = fmt.Sprintf("https://%s.%s/oauth2/token", options[0].Namespace, domain)
			mock.Subdomain = fmt.Sprintf("%s.%s", options[0].Namespace, domain)
		}
	}

	t.Logf("Deploying oauth2mock with IssuerURL: %s, TokenURL: %s, Subdomain: %s",
		mock.IssuerURL, mock.TokenURL, mock.Subdomain)
	return mock, startMock(t, mock, options...)
}

func startMock(t *testing.T, m *Mock, options ...Options) error {
	t.Helper()
	r := infrahelpers.ResourcesClient(t)

	namespace := "oauth2-mock"
	if len(options) > 0 && options[0].Namespace != "" {
		namespace = options[0].Namespace
	}

	require.NoError(t, infrahelpers.CreateNamespace(t, namespace))
	setup.DeclareCleanup(t, func() {
		t.Log("Cleaning up oauth2-mock")
		// No further cleanup is needed as the namespace will be deleted
		// as part of Namespace cleanup.
	})
	return m.start(t, r, options...)
}

func (m *Mock) start(t *testing.T, r *resources.Resources, options ...Options) error {
	err := m.parseTmpl()
	if err != nil {
		return err
	}

	namespace := "oauth2-mock"
	if len(options) > 0 && options[0].Namespace != "" {
		namespace = options[0].Namespace
	}

	require.NoError(t,
		decoder.DecodeEach(
			t.Context(),
			bytes.NewBuffer(m.parsedManifest),
			decoder.CreateHandler(r),
			decoder.MutateNamespace(namespace),
		),
	)

	return wait.For(conditions.New(r).DeploymentAvailable("mock-oauth2-server-deployment", namespace))
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
