package istio

import (
	"context"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/internal/webhooks"
	"github.com/kyma-project/istio/operator/pkg/labels"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type installArgs struct {
	client              client.Client
	istioCR             *operatorv1alpha2.Istio
	statusHandler       status.Status
	istioOperatorMerger istiooperator.Merger
	istioImageVersion   istiooperator.IstioImageVersion
	istioClient         libraryClient
}

func installIstio(ctx context.Context, args installArgs) (istiooperator.IstioImageVersion, described_errors.DescribedError) {

	istioImageVersion := args.istioImageVersion
	k8sClient := args.client
	istioCR := args.istioCR
	statusHandler := args.statusHandler
	iopMerger := args.istioOperatorMerger
	istioClient := args.istioClient

	ctrl.Log.Info("Starting Istio install", "istio version", istioImageVersion.Version())

	if _, ok := istioCR.Annotations[labels.LastAppliedConfiguration]; ok {
		lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
		if err != nil {
			ctrl.Log.Error(err, "Error evaluating Istio CR changes")
			return istioImageVersion, described_errors.NewDescribedError(err, "Istio install check failed")
		}

		if err := configuration.CheckIstioVersionUpdate(lastAppliedConfig.IstioTag, istioImageVersion.Tag()); err != nil {
			statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonIstioVersionUpdateNotAllowed))
			// We are already updating the condition, that's why we need to avoid another condition update by applying SetCondition(false)
			return istioImageVersion, described_errors.NewDescribedError(err, "Istio version update is not allowed").SetWarning().SetCondition(false)
		}
	}

	if !hasInstallationFinalizer(istioCR) {
		if err := addInstallationFinalizer(ctx, k8sClient, istioCR); err != nil {
			ctrl.Log.Error(err, "Failed to add Istio installation finalizer")
			return istioImageVersion, described_errors.NewDescribedError(err, "Could not add finalizer")
		}
	}

	clusterConfiguration, err := clusterconfig.EvaluateClusterConfiguration(ctx, k8sClient)
	if err != nil {
		return istioImageVersion, described_errors.NewDescribedError(err, "Could not evaluate cluster flavour")
	}

	clusterSize, err := clusterconfig.EvaluateClusterSize(context.Background(), k8sClient)
	if err != nil {
		ctrl.Log.Error(err, "Error occurred during evaluation of cluster size")
		return istioImageVersion, described_errors.NewDescribedError(err, "Could not evaluate cluster size")
	}

	ctrl.Log.Info("Installing Istio with", "profile", clusterSize.String())

	mergedIstioOperatorPath, err := iopMerger.Merge(clusterSize, istioCR, clusterConfiguration)
	if err != nil {
		statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonCustomResourceMisconfigured))
		return istioImageVersion, described_errors.NewDescribedError(err, "Could not merge Istio operator configuration").SetCondition(false)
	}

	err = istioClient.Install(mergedIstioOperatorPath)
	if err != nil {
		// In case of an error during the Istio installation, the old mutatingwebhook won't be deactivated, which will block later reconciliations
		err2 := webhooks.DeleteConflictedDefaultTag(ctx, k8sClient)
		if err2 != nil {
			ctrl.Log.Error(err2, "Error occurred when tried to clean conflicted webhooks")
		}

		return istioImageVersion, described_errors.NewDescribedError(err, "Could not install Istio")
	}

	err = addWardenValidationAndDisclaimer(ctx, k8sClient)
	if err != nil {
		return istioImageVersion, described_errors.NewDescribedError(err, "Could not add warden validation label")
	}

	err = gatherer.VerifyIstioPodsVersion(ctx, k8sClient, istioImageVersion.Version())
	if err != nil {
		return istioImageVersion, described_errors.NewDescribedError(err, "Verifying Pod versions in istio-system namespace failed")
	}

	ctrl.Log.Info("Istio installation succeeded")
	statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonIstioInstallSucceeded))

	if err := updateResourcesMetadataForSelector(ctx, k8sClient); err != nil {
		return istioImageVersion, described_errors.NewDescribedError(err, "could not update managed metadata")
	}

	return istioImageVersion, nil
}
