package testid

import (
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"testing"
)

type Options struct {
	Prefix string
}

func WithPrefix(prefix string) Option {
	return func(o *Options) {
		o.Prefix = prefix
	}
}

type Option func(*Options)

func CreateNamespaceWithRandomID(t *testing.T, options ...Option) (testId string, namespaceName string, err error) {
	t.Helper()
	opts := &Options{}
	for _, opt := range options {
		opt(opts)
	}

	testId = envconf.RandomName("test", 16)
	ns := testId
	if opts.Prefix != "" {
		ns = opts.Prefix + "-" + testId
	}

	t.Logf("Creating namespace %s", ns)
	return testId, ns, infrastructure.CreateNamespace(t, ns)
}
