package oauth2

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"testing"
	"text/template"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

//go:embed manifest.yaml
var rawManifest string

type Mock struct {
	IssuerURL, TokenURL string
	Subdomain           string

	parsedManifest []byte
}

func New(domain string) *Mock {
	return &Mock{
		IssuerURL: "http://mock-oauth2-server.oauth2-mock.svc.cluster.local",
		TokenURL:  fmt.Sprintf("https://oauth2-mock.%s/oauth2/token", domain),
		Subdomain: fmt.Sprintf("oauth2-mock.%s", domain),
	}
}

func (m *Mock) Start(ctx context.Context, r *resources.Resources) error {
	err := m.parseTmpl()
	if err != nil {
		return err
	}
	sbuf := bytes.NewBuffer(m.parsedManifest)
	err = decoder.DecodeEach(ctx, sbuf, decoder.CreateHandler(r))
	return wait.For(conditions.New(r).DeploymentAvailable("mock-oauth2-server-deployment", "oauth2-mock"))
}

func (m *Mock) Stop(ctx context.Context, r *resources.Resources) error {
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

// StartMock wraps around Mock.Start and declares cleanup after test finishes using Mock.Stop
func StartMock(t *testing.T, m *Mock, cfg *envconf.Config) error {
	r, err := resources.New(helpers.WrapTestLog(t, cfg.Client().RESTConfig()))
	if err != nil {
		return err
	}
	setup.DeclareCleanup(t, func() {
		t.Log("stopping oauth2-mock")
		require.NoError(t, m.Stop(setup.GetCleanupContext(), r))
	})
	t.Log("starting oauth2-mock")
	return m.Start(t.Context(), r)
}

func DeployOauth2Mock(domain string) types.EnvFunc {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		r, err := resources.New(cfg.Client().RESTConfig())
		mock := New(domain)
		err = mock.Start(ctx, r)
		if err != nil {
			return ctx, err
		}
		return context.WithValue(ctx, "oauth2-mock", mock), nil
	}
}

func DestroyOauth2Mock() types.EnvFunc {
	return func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
		mock, ok := FromContext(ctx)
		if !ok {
			return ctx, fmt.Errorf("oauth2-mock not found in context")
		}
		r, err := resources.New(cfg.Client().RESTConfig())
		if err != nil {
			return ctx, err
		}
		return ctx, mock.Stop(ctx, r)
	}
}

func FromContext(ctx context.Context) (*Mock, bool) {
	val, ok := ctx.Value("oauth2-mock").(*Mock)
	return val, ok
}
