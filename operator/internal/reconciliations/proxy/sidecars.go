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
	ProxyImageRepository string = "eu.gcr.io/kyma-project/external/istio/proxyv2"
)

// Reconcile runs Istio installation with merged Istio Operator manifest file when the trigger requires an installation.
func (s *Sidecars) Reconcile(ctx context.Context, client client.Client, logger *logr.Logger) error {
	expectedImage := pods.SidecarImage{Repository: ProxyImageRepository, Tag: fmt.Sprintf("%s-%s", s.IstioVersion, s.IstioImageBase)}

	warnings, err := sidecars.ProxyReset(ctx, client, expectedImage, s.CniEnabled, logger)
	if err != nil {
		return err
	}
	logger.Info("Proxy reset processed for:", warnings)

	return nil
}
