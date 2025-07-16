package testid

import (
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"testing"
)

type Options struct {
	Prefix string
}

func CreateNamespaceWithRandomID(t *testing.T, options ...Options) (testId string, namespaceName string, err error) {
	t.Helper()
	testId = envconf.RandomName("test", 16)
	ns := testId
	if len(options) > 0 && options[0].Prefix != "" {
		ns = options[0].Prefix + "-" + testId
	}

	t.Logf("Creating namespace %s", ns)
	return testId, ns, infrastructure.CreateNamespace(t, ns)
}
