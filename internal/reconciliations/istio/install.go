package istio

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/internal/images"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/describederrors"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/internal/webhooks"
	"github.com/kyma-project/istio/operator/pkg/labels"
)

type installArgs struct {
	client              client.Client
	istioCR             *operatorv1alpha2.Istio
	statusHandler       status.Status
	istioOperatorMerger istiooperator.Merger
	istioImageVersion   istiooperator.IstioImageVersion
	istioClient         libraryClient
	istioImages         images.Images
}

//nolint:funlen // Function 'installIstio' has too many statements (51 > 50) TODO: refactor.
func installIstio(ctx context.Context, args installArgs) (istiooperator.IstioImageVersion, describederrors.DescribedError) {
	istioImageVersion := args.istioImageVersion
	k8sClient := args.client
	istioCR := args.istioCR
	statusHandler := args.statusHandler
	iopMerger := args.istioOperatorMerger
	istioClient := args.istioClient
	istioImages := args.istioImages

	ctrl.Log.Info("Starting Istio install", "istio version", istioImageVersion.Version())

	lastAppliedConfig := configuration.AppliedConfig{}
	if _, ok := istioCR.Annotations[labels.LastAppliedConfiguration]; ok {
		var err error
		lastAppliedConfig, err = configuration.GetLastAppliedConfiguration(istioCR)
		if err != nil {
			ctrl.Log.Error(err, "Error evaluating Istio CR changes")
			return istioImageVersion, describederrors.NewDescribedError(err, "Istio install check failed")
		}

		if err = configuration.CheckIstioVersionUpdate(lastAppliedConfig.IstioTag, istioImageVersion.Tag()); err != nil {
			statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonIstioVersionUpdateNotAllowed))
			// We are already updating the condition, that's why we need to avoid another condition update by applying SetCondition(false)
			return istioImageVersion, describederrors.NewDescribedError(err, "Istio version update is not allowed").SetWarning().SetCondition(false)
		}
	}

	if !hasInstallationFinalizer(istioCR) {
		if err := addInstallationFinalizer(ctx, k8sClient, istioCR); err != nil {
			ctrl.Log.Error(err, "Failed to add Istio installation finalizer")
			return istioImageVersion, describederrors.NewDescribedError(err, "Could not add finalizer")
		}
	}

	// Check the cluster provider for the cluster configuration annotation purposes
	clusterProvider, err := clusterconfig.GetClusterProvider(ctx, k8sClient)
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not determine cluster provider")
	}

	clusterConfiguration, err := clusterconfig.EvaluateClusterConfiguration(ctx, k8sClient, clusterProvider)
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not evaluate cluster flavour")
	}

	enableDualStack, err := clusterconfig.IsDualStackEnabled(ctx, k8sClient)
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not evaluate if dual stack is enabled")
	}
	if enableDualStack {
		ctrl.Log.Info("Istio is running with IPDualStack enabled")
	}

	clusterSize, err := clusterconfig.EvaluateClusterSize(context.Background(), k8sClient)
	if err != nil {
		ctrl.Log.Error(err, "Error occurred during evaluation of cluster size")
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not evaluate cluster size")
	}

	// Gateway API CRD reconciliation process

	// Short-circuit evaluation
	// Istio CR - enableGatewayAPI: true
	// Explicitly Gateway API CRD management enabled
	gatewayAPIEnabled := istioCR.Spec.Experimental != nil &&
		istioCR.Spec.Experimental.EnableGatewayAPI != nil &&
		*istioCR.Spec.Experimental.EnableGatewayAPI

	// Istio CR - enableGatewayAPI: false or lack of this field
	// Explicitly Gateway API CRD management disabled when previously enabled, so Gateway API CRDs exist in the cluster
	gatewayAPIDisabled := lastAppliedConfig.Experimental != nil &&
		lastAppliedConfig.Experimental.EnableAlphaGatewayAPI == true &&
		istioCR.Spec.Experimental != nil && (istioCR.Spec.Experimental.EnableGatewayAPI == nil ||
		!*istioCR.Spec.Experimental.EnableGatewayAPI)

	gatewayAPICRDManager := NewGatewayAPICRDManager(k8sClient)
	if gatewayAPIEnabled {
		ctrl.Log.Info("Installing Gateway API CRDs (enabled via spec.experimental.enableGatewayAPI)")
		result, err := gatewayAPICRDManager.Install(ctx)
		if err != nil {
			ctrl.Log.Error(err, "Gateway API CRDs installation failed", "istioVersion", istioImageVersion.Version())
			return istioImageVersion, describederrors.NewDescribedError(err, "Could not install Gateway API CRDs")
		}
		if result.HasUnmanagedCRDs() {
			ctrl.Log.Info("Some Gateway API CRDs exist on the cluster but are not managed by the Istio module – they were skipped",
				"unmanagedCRDs", result.UnmanagedCRDs,
				"action", fmt.Sprintf("add label %s=%s to each listed CRD to allow the Istio module to manage it", labels.ModuleLabelKey, labels.ModuleLabelValue),
			)
		}
		ctrl.Log.Info("Gateway API CRDs reconciled, proceeding with Istio installation",
			"created", len(result.CreatedCRDs),
			"updated", len(result.UpdatedCRDs),
			"unchanged", len(result.UnchangedCRDs),
			"unmanagedSkipped", len(result.UnmanagedCRDs),
		)
	} else if gatewayAPIDisabled {
		// Reconciliation: enableGatewayAPI is explicitly set to false or lacks this field in Istio CR, when previously explicitly enabled.
		// Clean up any module-owned CRDs that remain from a previous enablement.
		// Check for the blocking Gateway API CRs
		ctrl.Log.Info("Gateway API CRD feature explicitly disabled – removing labeled CRDs if present")
		if err := gatewayAPICRDManager.Uninstall(ctx, statusHandler, istioCR); err != nil {
			ctrl.Log.Error(err, "Failed to remove Gateway API CRDs, but continuing", "note", "Manual cleanup may be required")
			return istioImageVersion, describederrors.NewDescribedError(err,
				"Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning").
				DisableErrorWrap().
				SetWarning().
				SetCondition(false)
		}

		ctrl.Log.Info("Gateway API CRDs cleanup completed successfully")
	}

	ctrl.Log.Info("Installing Istio with", "profile", clusterSize.String())
	var options []operatorv1alpha2.MergeOption
	if enableDualStack {
		options = append(options, operatorv1alpha2.WithDualStackEnabled())
	}
	mergedIstioOperatorPath, err := iopMerger.Merge(clusterSize, istioCR, clusterConfiguration, istioImages, options...)
	if err != nil {
		statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonCustomResourceMisconfigured))
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not merge Istio operator configuration").SetCondition(false)
	}

	err = istioClient.Install(mergedIstioOperatorPath)
	if err != nil {
		// In case of an error during the Istio installation, the old mutatingwebhook won't be deactivated, which will block later reconciliations
		err2 := webhooks.DeleteConflictedDefaultTag(ctx, k8sClient)
		if err2 != nil {
			ctrl.Log.Error(err2, "Error occurred when tried to clean conflicted webhooks")
		}

		return istioImageVersion, describederrors.NewDescribedError(err, "Could not install Istio")
	}

	err = addWardenValidationAndDisclaimer(ctx, k8sClient)
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not add warden validation label")
	}

	if err = patchModuleResourcesWithModuleLabel(ctx, k8sClient); err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "could not update managed metadata")
	}

	err = gatherer.VerifyIstioPodsVersion(ctx, k8sClient, istioImageVersion.Version())
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Verifying Pod versions in istio-system namespace failed")
	}

	ctrl.Log.Info("Istio installation succeeded")
	statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonIstioInstallSucceeded))

	return istioImageVersion, nil
}
