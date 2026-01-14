package client

import (
	"net/http"
	"sync/atomic"
	"testing"

	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	securityv1 "istio.io/client-go/pkg/apis/security/v1"
	"istio.io/client-go/pkg/apis/security/v1beta1"
	telemetryv1 "istio.io/client-go/pkg/apis/telemetry/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/e2e-framework/klient/conf"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/envconf"

	"github.com/kyma-project/istio/operator/api/v1alpha2"

	"k8s.io/client-go/rest"

	httphelper "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/http"
)

const KubernetesClientLogPrefix = "kube-client"

var isInitialized atomic.Bool

func GetKubeConfig(t *testing.T) *rest.Config {
	t.Helper()
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)
	return wrapTestLog(t, cfg.Client().RESTConfig())
}

func ResourcesClient(t *testing.T) (*resources.Resources, error) {
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)

	r, err := resources.New(wrapTestLog(t, cfg.Client().RESTConfig()))
	if err != nil {
		t.Logf("Failed to create resources client: %v", err)
		return nil, err
	}

	if !isInitialized.Load() {
		err = v1alpha2.AddToScheme(r.GetScheme())
		if err != nil {
			t.Logf("Failed to add v1alpha2 scheme: %v", err)
			return nil, err
		}

		err = v1alpha3.AddToScheme(r.GetScheme())
		if err != nil {
			t.Logf("Failed to add v1alpha3 scheme: %v", err)
			return nil, err
		}

		err = v1beta1.AddToScheme(r.GetScheme())
		if err != nil {
			t.Logf("Failed to add v1beta1 scheme: %v", err)
			return nil, err
		}

		err = telemetryv1.AddToScheme(r.GetScheme())
		if err != nil {
			t.Logf("Failed to add Istio telemetry v1 scheme: %v", err)
			return nil, err
		}

		err = securityv1.AddToScheme(r.GetScheme())
		if err != nil {
			t.Logf("Failed to add Istio security v1 scheme: %v", err)
			return nil, err
		}

		err = networkingv1.AddToScheme(r.GetScheme())
		if err != nil {
			t.Logf("Failed to add Istio networking v1 scheme: %v", err)
			return nil, err
		}

		isInitialized.Store(true)
	}

	return r, nil
}

func wrapTestLog(t *testing.T, cfg *rest.Config) *rest.Config {
	cfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return httphelper.TestLogTransportWrapper(t, KubernetesClientLogPrefix, "", nil, rt)
	})
	return cfg
}

func GetClientSet(t *testing.T) (*kubernetes.Clientset, error) {
	t.Helper()
	restConfig, err := config.GetConfig()
	if err != nil {
		t.Logf("Could not create in-cluster config: err=%s", err)
		return nil, err
	}
	return kubernetes.NewForConfig(restConfig)
}
