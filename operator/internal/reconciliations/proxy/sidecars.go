package proxy

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Sidecars struct {
	IstioVersion   string
	IstioImageBase string
	CniEnabled     bool
}

const (
	imageRepository string = "eu.gcr.io/kyma-project/external/istio/proxyv2"
)

// Reconcile runs Proxy Reset action, which checks if any of sidecars need a restart and proceed with rollout.
func (s *Sidecars) Reconcile(ctx context.Context, client client.Client, logger logr.Logger) error {
	expectedImage := pods.SidecarImage{Repository: imageRepository, Tag: fmt.Sprintf("%s-%s", s.IstioVersion, s.IstioImageBase)}
	logger.Info("Running proxy sidecar reset", "expected image", expectedImage)

	warnings, err := sidecars.ProxyReset(ctx, client, expectedImage, s.CniEnabled, &logger)
	if err != nil {
		return err
	}
	if len(warnings) > 0 {
		for _, w := range warnings {
			logger.Info("Proxy reset warning:", "name", w.Name, "namespace", w.Namespace, "kind", w.Kind, "message", w.Message)
		}
	}

	return nil
}
