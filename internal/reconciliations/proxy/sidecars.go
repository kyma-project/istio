package proxy

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Sidecars struct {
	IstioVersion   string
	IstioImageBase string
	Log            logr.Logger
	Client         client.Client
	Merger         manifest.Merger
}

const (
	// TODO Update image repository to new one
	imageRepository string = "eu.gcr.io/kyma-project/external/istio/proxyv2"
)

// Reconcile runs Proxy Reset action, which checks if any of sidecars need a restart and proceed with rollout.
func (s *Sidecars) Reconcile(ctx context.Context, istioCr v1alpha1.Istio) error {
	expectedImage := pods.SidecarImage{Repository: imageRepository, Tag: fmt.Sprintf("%s-%s", s.IstioVersion, s.IstioImageBase)}
	s.Log.Info("Running proxy sidecar reset", "expected image", expectedImage)

	version, err := gatherer.GetIstioPodsVersion(ctx, s.Client)
	if err != nil {
		return err
	}

	if s.IstioVersion != version {
		return fmt.Errorf("istio-system pods version: %s do not match target version: %s", version, s.IstioVersion)
	}

	cSize, err := clusterconfig.EvaluateClusterSize(context.Background(), s.Client)
	if err != nil {
		s.Log.Error(err, "Error occurred during evaluation of cluster size")
		return err
	}

	ctrl.Log.Info("Installing istio with", "profile", cSize.String())

	iop, err := s.Merger.GetIstioOperator(cSize.DefaultManifestPath())
	if err != nil {
		return err
	}

	expectedResources, err := istioCr.GetProxyResources(iop)
	if err != nil {
		return err
	}

	warnings, err := sidecars.ProxyReset(ctx, s.Client, expectedImage, expectedResources, &s.Log)
	if err != nil {
		return err
	}
	if len(warnings) > 0 {
		for _, w := range warnings {
			s.Log.Info("Proxy reset warning:", "name", w.Name, "namespace", w.Namespace, "kind", w.Kind, "message", w.Message)
		}
	}

	return nil
}
