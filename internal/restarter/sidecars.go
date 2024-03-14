package restarter

import (
	"context"
	"fmt"
	"strings"

	"github.com/coreos/go-semver/semver"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/pkg/errors"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const errorDescription = "Error occurred during reconciliation of Istio Sidecars"

type SidecarsRestarter struct {
	Log           logr.Logger
	Client        client.Client
	Merger        manifest.Merger
	ProxyResetter sidecars.ProxyResetter
	Predicates    []filter.SidecarProxyPredicate
	StatusHandler status.Status
}

func NewSidecarsRestarter(logger logr.Logger, client client.Client, merger manifest.Merger, resetter sidecars.ProxyResetter, predicates []filter.SidecarProxyPredicate, statusHandler status.Status) *SidecarsRestarter {
	return &SidecarsRestarter{
		Log:           logger,
		Client:        client,
		Merger:        merger,
		ProxyResetter: resetter,
		Predicates:    predicates,
		StatusHandler: statusHandler,
	}
}

const (
	imageRepository string = "europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2"
)

// Restart runs Proxy Reset action, which checks if any of sidecars need a restart and proceed with rollout.
func (s *SidecarsRestarter) Restart(ctx context.Context, istioCR *v1alpha2.Istio) described_errors.DescribedError {
	clusterSize, err := clusterconfig.EvaluateClusterSize(ctx, s.Client)
	if err != nil {
		s.Log.Error(err, "Error occurred during evaluation of cluster size")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription)
	}

	ctrl.Log.Info("Istio proxy resetting with", "profile", clusterSize.String())
	iop, err := s.Merger.GetIstioOperator(clusterSize.GetManifestPath())
	if err != nil {
		s.Log.Error(err, "Failed to get IstioOperator")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription)
	}

	v, err := semver.NewVersion(iop.Spec.Tag.GetStringValue())
	if err != nil {
		ctrl.Log.Error(err, "Error occurred during parsing Istio semver version")
		return described_errors.NewDescribedError(err, "Could not parse Istio semver version")
	}

	istioVersion := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)

	expectedImage := pods.NewSidecarImage(iop.Spec.Hub, iop.Spec.Tag.GetStringValue())
	s.Log.Info("Running proxy sidecar reset", "expected image", expectedImage)

	version, err := gatherer.GetIstioPodsVersion(ctx, s.Client)
	if err != nil {
		s.Log.Error(err, "Failed to get istio pod version")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription)
	}

	if istioVersion != version {
		err := fmt.Errorf("istio-system pods version: %s do not match target version: %s", version, istioVersion)
		s.Log.Error(err, err.Error())
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription)
	}

	expectedResources, err := istioCR.GetProxyResources(iop)
	if err != nil {
		s.Log.Error(err, "Failed to get Istio Proxy resources")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription)
	}

	warnings, err := s.ProxyResetter.ProxyReset(ctx, s.Client, expectedImage, expectedResources, s.Predicates, &s.Log)
	if err != nil {
		s.Log.Error(err, "Failed to reset proxy")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription)
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
		warningErr := described_errors.NewDescribedError(errors.New("Istio controller could not restart one or more istio-injected pods."), "Not all pods with Istio injection could be restarted. Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning").SetWarning()
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarManualRestartRequired, warningMessage))
		s.Log.Info(warningMessage)
		return warningErr
	}

	s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartSucceeded))
	return nil
}
