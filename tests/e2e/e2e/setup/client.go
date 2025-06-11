package setup

import (
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
)

// ClientFromKubeconfig creates a Kubernetes client based as in config.GetConfig()
// a logger needs to be provided, that will log requests going to the Kubernetes API server.
func ClientFromKubeconfig(t *testing.T) (client.Client, error) {
	k8sConfig, err := config.GetConfig()
	if err != nil {
		return nil, err
	}

	k8sConfig.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return &loggingRoundTripper{
			rt: rt,
			t:  t,
		}
	})

	k8sClient, err := client.New(k8sConfig, client.Options{})
	return k8sClient, nil
}

type loggingRoundTripper struct {
	rt http.RoundTripper
	t  *testing.T
}

const KubernetesClientLogPrefix = "[K8S] "

// RoundTrip implements the http.RoundTripper interface to log requests and responses.
//
// NOTE: Current implementation does not include retry logic, but it can be extended to do so.
func (l *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	l.t.Logf(KubernetesClientLogPrefix+"Request to API Server: %s %s", req.Method, req.URL)

	resp, err := l.rt.RoundTrip(req)
	if err != nil {
		l.t.Errorf(KubernetesClientLogPrefix+"Request to API Server failed: %s %s", req.URL, err.Error())
		return nil, err
	}
	l.t.Logf(KubernetesClientLogPrefix+"Response from API Server: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	return resp, nil
}
