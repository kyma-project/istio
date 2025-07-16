package infrastructure

import (
	"net/http"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"testing"

	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
	"k8s.io/client-go/rest"
)

const KubernetesClientLogPrefix = "kube-client"

func ResourcesClient(t *testing.T) (*resources.Resources, error) {
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)

	return resources.New(wrapTestLog(t, cfg.Client().RESTConfig()))
}

func wrapTestLog(t *testing.T, cfg *rest.Config) *rest.Config {
	cfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return httphelper.TestLogTransportWrapper(t, KubernetesClientLogPrefix, rt)
	})
	return cfg
}
