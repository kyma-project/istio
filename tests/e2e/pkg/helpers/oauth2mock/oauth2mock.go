package oauth2mock

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"testing"
	"text/template"

	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
)

//go:embed manifest.yaml
var rawManifest string

type Mock struct {
	IssuerURL, TokenURL string
	Subdomain           string

	parsedManifest []byte
}

func DeployMock(t *testing.T, domain string) (*Mock, error) {
	t.Helper()
	mock := &Mock{
		IssuerURL: "mock-oauth2-server.oauth2-mock.svc.cluster.local",
		TokenURL:  fmt.Sprintf("https://oauth2-mock.%s/oauth2/token", domain),
		Subdomain: fmt.Sprintf("oauth2-mock.%s", domain),
	}

	return mock, startMock(t, mock)
}

func startMock(t *testing.T, m *Mock) error {
	t.Helper()
	r := infrahelpers.ResourcesClient(t)
	setup.DeclareCleanup(t, func() {
		t.Log("stopping oauth2-mock")
		require.NoError(t, m.stop(setup.GetCleanupContext(), r))
	})
	t.Log("starting oauth2-mock")
	return m.start(t, r)
}

func (m *Mock) start(t *testing.T, r *resources.Resources) error {
	err := m.parseTmpl()
	if err != nil {
		return err
	}

	require.NoError(t,
		decoder.DecodeEach(
			t.Context(),
			bytes.NewBuffer(m.parsedManifest),
			decoder.CreateHandler(r),
		),
	)

	return wait.For(conditions.New(r).DeploymentAvailable("mock-oauth2-server-deployment", "oauth2-mock"))
}

func (m *Mock) stop(ctx context.Context, r *resources.Resources) error {
	return r.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "oauth2-mock"}})
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
