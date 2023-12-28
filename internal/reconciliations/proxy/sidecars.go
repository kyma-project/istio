package proxy

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type SidecarsReconciliation interface {
	Reconcile(ctx context.Context, istioCr v1alpha1.Istio) (string, error)
	AddReconcilePredicate(predicate filter.SidecarProxyPredicate)
}

type Sidecars struct {
	IstioVersion   string
	IstioImageBase string
	Log            logr.Logger
	Client         client.Client
	Merger         manifest.Merger
	ProxyResetter  sidecars.ProxyResetter
	Predicates     []filter.SidecarProxyPredicate
}

const (
	imageRepository string = "europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2"
)

// Reconcile runs Proxy Reset action, which checks if any of sidecars need a restart and proceed with rollout.
func (s *Sidecars) Reconcile(ctx context.Context, istioCr v1alpha1.Istio) (string, error) {
	expectedImage := pods.SidecarImage{Repository: imageRepository, Tag: fmt.Sprintf("%s-%s", s.IstioVersion, s.IstioImageBase)}
	s.Log.Info("Running proxy sidecar reset", "expected image", expectedImage)

	version, err := gatherer.GetIstioPodsVersion(ctx, s.Client)
	if err != nil {
		return "", err
	}

	if s.IstioVersion != version {
		return "", fmt.Errorf("istio-system pods version: %s do not match target version: %s", version, s.IstioVersion)
	}

	cSize, err := clusterconfig.EvaluateClusterSize(context.Background(), s.Client)
	if err != nil {
		s.Log.Error(err, "Error occurred during evaluation of cluster size")
		return "", err
	}

	ctrl.Log.Info("Istio proxy resetting with", "profile", cSize.String())
	iop, err := s.Merger.GetIstioOperator(cSize.DefaultManifestPath())
	if err != nil {
		return "", err
	}
	expectedResources, err := istioCr.GetProxyResources(iop)
	if err != nil {
		return "", err
	}

	warnings, err := s.ProxyResetter.ProxyReset(ctx, s.Client, expectedImage, expectedResources, s.Predicates, &s.Log)
	if err != nil {
		return "", err
	}
	warningsCount := len(warnings)
	if warningsCount > 0 {
		podsLimit := 5
		pods := []string{}
		for _, w := range warnings {
			if podsLimit--; podsLimit >= 0 {
				pods = append(pods, fmt.Sprintf("%s/%s", w.Namespace, w.Name))
			}
			s.Log.Info("Proxy reset warning:", "name", w.Name, "namespace", w.Namespace, "kind", w.Kind, "message", w.Message)
		}
		warningMessage := fmt.Sprintf("The sidecars of the following workloads could not be restarted: %s",
			strings.Join(pods, ", "))
		if warningsCount-len(pods) > 0 {
			warningMessage += fmt.Sprintf(" and %d additional workload(s)", warningsCount-len(pods))
		}
		return warningMessage, nil
	}

	return "", nil
}

func (s *Sidecars) AddReconcilePredicate(predicate filter.SidecarProxyPredicate) {
	s.Predicates = append(s.Predicates, predicate)
}
