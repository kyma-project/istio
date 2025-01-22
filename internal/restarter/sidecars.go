package restarter

import (
	"context"
	"fmt"
	"strings"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/pkg/errors"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const errorDescription = "Error occurred during reconciliation of Istio Sidecars"

type SidecarRestarter struct {
	Log            logr.Logger
	Client         client.Client
	Merger         istiooperator.Merger
	ProxyRestarter sidecars.ProxyRestarter
	StatusHandler  status.Status
}

func NewSidecarsRestarter(logger logr.Logger, client client.Client, merger istiooperator.Merger, proxyRestarter sidecars.ProxyRestarter, statusHandler status.Status) *SidecarRestarter {
	return &SidecarRestarter{
		Log:            logger,
		Client:         client,
		Merger:         merger,
		ProxyRestarter: proxyRestarter,
		StatusHandler:  statusHandler,
	}
}

// Restart runs Proxy Reset action, which checks if any of sidecars need a restart and proceed with rollout.
func (s *SidecarRestarter) Restart(ctx context.Context, istioCR *v1alpha2.Istio) (described_errors.DescribedError, bool) {
	clusterSize, err := clusterconfig.EvaluateClusterSize(ctx, s.Client)
	if err != nil {
		s.Log.Error(err, "Error occurred during evaluation of cluster size")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription), false
	}

	ctrl.Log.Info("Istio proxy resetting with", "profile", clusterSize.String())
	iop, err := s.Merger.GetIstioOperator(clusterSize)
	if err != nil {
		s.Log.Error(err, "Failed to get IstioOperator")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription), false
	}

	istioImageVersion, err := s.Merger.GetIstioImageVersion()
	if err != nil {
		ctrl.Log.Error(err, "Error getting Istio version from istio operator file")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, "Could not get Istio version from istio operator file"), false
	}

	tag, ok := iop.Spec.Tag.(string)
	if !ok {
		ctrl.Log.Error(err, "Error getting Istio tag from istio operator file")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, "Could not get Istio tag from istio operator file"), false
	}

	expectedImage := predicates.NewSidecarImage(iop.Spec.Hub, tag)
	s.Log.Info("Running proxy sidecar reset", "expected image", expectedImage)

	err = gatherer.VerifyIstioPodsVersion(ctx, s.Client, istioImageVersion.Version())
	if err != nil {
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, "Verifying Pod versions in istio-system namespace failed"), false
	}

	expectedResources, err := istioCR.GetProxyResources(iop)
	if err != nil {
		s.Log.Error(err, "Failed to get Istio Proxy resources")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription), false
	}

	warnings, hasMorePods, err := s.ProxyRestarter.RestartProxies(ctx, s.Client, expectedImage, expectedResources, istioCR, &s.Log)
	if err != nil {
		s.Log.Error(err, "Failed to reset proxy")
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartFailed))
		return described_errors.NewDescribedError(err, errorDescription), false
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
		return warningErr, false
	}

	if !hasMorePods {
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartSucceeded))
	} else {
		s.StatusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonProxySidecarRestartPartiallySucceeded))
	}

	return nil, hasMorePods
}
