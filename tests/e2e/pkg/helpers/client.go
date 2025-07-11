package helpers

import (
	"net/http"
	"testing"

	"k8s.io/client-go/rest"
)

const KubernetesClientLogPrefix = "kube-client"

func WrapTestLog(t *testing.T, cfg *rest.Config) *rest.Config {
	cfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return TestLogTransportWrapper(t, KubernetesClientLogPrefix, rt)
	})
	return cfg
}
