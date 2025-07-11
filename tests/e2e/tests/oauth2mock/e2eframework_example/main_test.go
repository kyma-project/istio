package e2eframework_example

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup/oauth2"
	"sigs.k8s.io/e2e-framework/klient/decoder"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/pkg/env"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/envfuncs"
	"sigs.k8s.io/e2e-framework/third_party/k3d"
)

const IstioModuleManifest = "https://github.com/kyma-project/istio/releases/download/1.20.1/istio-manager.yaml"

var (
	testEnv env.Environment
)

func TestMain(m *testing.M) {
	testEnv = env.New()
	runID := envconf.RandomName("test", 16)

	testEnv.Setup(
		envfuncs.CreateCluster(k3d.NewProvider(), "k3d-"+runID),
		envfuncs.CreateNamespace("kyma-system"),
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			r, err := resources.New(cfg.Client().RESTConfig())
			resp, err := http.Get(IstioModuleManifest)
			if err != nil {
				return ctx, err
			}
			defer resp.Body.Close()
			err = decoder.DecodeEach(ctx, resp.Body, decoder.CreateHandler(r), decoder.MutateNamespace("kyma-system"))
			if err != nil {
				return ctx, err
			}
			return ctx, nil
		},
		envfuncs.CreateNamespace(runID),
		oauth2.DeployOauth2Mock("local.kyma.dev"),
	)
	testEnv.Finish(
		oauth2.DestroyOauth2Mock(),
		envfuncs.DeleteNamespace(runID),
		func(ctx context.Context, cfg *envconf.Config) (context.Context, error) {
			r, err := resources.New(cfg.Client().RESTConfig())
			resp, err := http.Get(IstioModuleManifest)
			if err != nil {
				return ctx, err
			}
			defer resp.Body.Close()
			err = decoder.DecodeEach(ctx, resp.Body, decoder.DeleteHandler(r), decoder.MutateNamespace("kyma-system"))
			if err != nil {
				return ctx, err
			}
			return ctx, nil
		},
		envfuncs.DeleteNamespace("kyma-system"),
		envfuncs.DestroyCluster("k3d-"+runID),
	)
	os.Exit(testEnv.Run(m))
}
