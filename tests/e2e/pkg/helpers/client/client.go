package client

import (
	"net/http"
	"sync/atomic"
	"testing"

	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
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

func ResourcesClientFromConfig() (*resources.Resources, error) {
	path := conf.ResolveKubeConfigFile()
	cfg := envconf.NewWithKubeConfig(path)

	r, err := resources.New(cfg.Client().RESTConfig())
	if err != nil {
		return nil, err
	}

	if err := registerSchemes(r); err != nil {
		return nil, err
	}

	return r, nil
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
		if err := registerSchemes(r); err != nil {
			t.Logf("Failed to register schemes: %v", err)
			return nil, err
		}
		isInitialized.Store(true)
	}

	return r, nil
}

func registerSchemes(r *resources.Resources) error {
	if err := v1alpha2.AddToScheme(r.GetScheme()); err != nil {
		return err
	}
	if err := v1alpha3.AddToScheme(r.GetScheme()); err != nil {
		return err
	}
	if err := v1beta1.AddToScheme(r.GetScheme()); err != nil {
		return err
	}
	if err := telemetryv1.AddToScheme(r.GetScheme()); err != nil {
		return err
	}
	if err := securityv1.AddToScheme(r.GetScheme()); err != nil {
		return err
	}
	if err := networkingv1.AddToScheme(r.GetScheme()); err != nil {
		return err
	}
	return nil
}

func wrapTestLog(t *testing.T, cfg *rest.Config) *rest.Config {
	artifact := httphelper.OpenTestArtifactLog(t, KubernetesClientLogPrefix)
	cfg.Wrap(func(rt http.RoundTripper) http.RoundTripper {
		return httphelper.TestLogTransportWrapper(t, KubernetesClientLogPrefix, "", nil, rt,
			httphelper.SuppressTestLog(),
			httphelper.WithOutput(artifact),
		)
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
