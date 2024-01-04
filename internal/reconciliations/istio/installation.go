package istio

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/internal/webhooks"
	"github.com/thoas/go-funk"
	"k8s.io/client-go/util/retry"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/internal/resources"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	sidecarRemover "github.com/kyma-project/istio/operator/pkg/lib/sidecars/remove"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type InstallationReconciliation interface {
	Reconcile(ctx context.Context, istioCR *operatorv1alpha1.Istio, statusHandler status.Status, istioResourceListPath string) described_errors.DescribedError
}

type Installation struct {
	IstioClient    LibraryClient
	IstioVersion   string
	IstioImageBase string
	Client         client.Client
	Merger         manifest.Merger
}

const (
	LastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
	installationFinalizer    string = "istios.operator.kyma-project.io/istio-installation"
)

// Reconcile runs Istio reconciliation to install, upgrade or uninstall Istio and returns the updated Istio CR.
func (i *Installation) Reconcile(ctx context.Context, istioCR *operatorv1alpha1.Istio, statusHandler status.Status, istioResourceListPath string) described_errors.DescribedError {
	istioTag := fmt.Sprintf("%s-%s", i.IstioVersion, i.IstioImageBase)

	shouldInstallIstio, err := shouldInstall(istioCR, istioTag)

	if err != nil {
		ctrl.Log.Error(err, "Error evaluating Istio CR changes")
		return described_errors.NewDescribedError(err, "Istio install check failed")
	}

	if shouldInstallIstio {
		ctrl.Log.Info("Starting Istio install", "istio version", i.IstioVersion, "istio image", i.IstioImageBase)

		if !hasInstallationFinalizer(istioCR) {
			if err = addInstallationFinalizer(ctx, i.Client, istioCR); err != nil {
				ctrl.Log.Error(err, "Failed to add Istio installation finalizer")
				return described_errors.NewDescribedError(err, "Could not add finalizer")
			}
		}

		clusterConfiguration, err := clusterconfig.EvaluateClusterConfiguration(ctx, i.Client)
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not evaluate cluster flavour")
		}

		// As we define default IstioOperator values in a templated manifest, we need to apply the istio version and values from
		// Istio CR to this default configuration to get the final IstoOperator that is used for installing and updating Istio.
		templateData := manifest.TemplateData{IstioVersion: i.IstioVersion, IstioImageBase: i.IstioImageBase}

		cSize, err := clusterconfig.EvaluateClusterSize(context.Background(), i.Client)
		if err != nil {
			ctrl.Log.Error(err, "Error occurred during evaluation of cluster size")
			return described_errors.NewDescribedError(err, "Could not evaluate cluster size")
		}

		ctrl.Log.Info("Installing Istio with", "profile", cSize.String())

		mergedIstioOperatorPath, err := i.Merger.Merge(cSize.DefaultManifestPath(), istioCR, templateData, clusterConfiguration)
		if err != nil {
			statusHandler.SetCondition(istioCR, operatorv1alpha1.NewReasonWithMessage(operatorv1alpha1.ConditionReasonCustomResourceMisconfigured))
			return described_errors.NewDescribedError(err, "Could not merge Istio operator configuration")
		}

		err = i.IstioClient.Install(mergedIstioOperatorPath)
		if err != nil {
			// In case of an error during the Istio installation, the old mutatingwebhook won't be deactivated, which will block later reconciliations
			err2 := webhooks.DeleteConflictedDefaultTag(ctx, i.Client)
			if err2 != nil {
				ctrl.Log.Error(err2, "Error occurred when tried to clean conflicted webhooks")
			}

			return described_errors.NewDescribedError(err, "Could not install Istio")
		}

		err = addWardenValidationAndDisclaimer(ctx, i.Client)
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not add warden validation label")
		}

		version, err := gatherer.GetIstioPodsVersion(ctx, i.Client)
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not get Istio sidecar version on cluster")
		}

		if i.IstioVersion != version {
			return described_errors.NewDescribedError(fmt.Errorf("istio-system pods version: %s do not match target version: %s", version, i.IstioVersion), "Istio installation failed")
		}

		ctrl.Log.Info("Istio installation succeeded")
		statusHandler.SetCondition(istioCR, operatorv1alpha1.NewReasonWithMessage(operatorv1alpha1.ConditionReasonIstioInstallSucceeded))

		err = restartIngressGatewayIfNeeded(ctx, i.Client, istioCR)
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not restart Istio Ingress GW deployment")
		}

		// We use the installation finalizer to track if the deletion was already executed so can make the uninstallation process more reliable.
	} else if shouldDelete(istioCR) && hasInstallationFinalizer(istioCR) {
		ctrl.Log.Info("Starting Istio uninstall")

		istioResourceFinder, err := resources.NewIstioResourcesFinderFromConfigYaml(ctx, i.Client, ctrl.Log, istioResourceListPath)
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not read customer resources finder configuration")
		}

		clientResources, err := istioResourceFinder.FindUserCreatedIstioResources()
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not get customer resources from the cluster")
		}

		if len(clientResources) > 0 {
			funk.ForEach(clientResources, func(a resources.Resource) {
				ctrl.Log.Info("Customer resource is blocking Istio deletion", a.GVK.Kind, fmt.Sprintf("%s/%s", a.Namespace, a.Name))
			})
			statusHandler.SetCondition(istioCR, operatorv1alpha1.NewReasonWithMessage(operatorv1alpha1.ConditionReasonIstioCRsDangling))
			return described_errors.NewDescribedError(fmt.Errorf("could not delete Istio module instance since there are %d customer resources present", len(clientResources)),
				"There are Istio resources that block deletion. Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning").DisableErrorWrap().SetWarning()
		}

		err = i.IstioClient.Uninstall(ctx)
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not uninstall istio")
		}

		warnings, err := sidecarRemover.RemoveSidecars(ctx, i.Client, &ctrl.Log)
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not remove istio sidecars")
		}

		if len(warnings) > 0 {
			for _, w := range warnings {
				ctrl.Log.Info("Removing sidecar warning:", "name", w.Name, "namespace", w.Namespace, "kind", w.Kind, "message", w.Message)
			}
		}

		ctrl.Log.Info("Istio uninstall succeeded")
		statusHandler.SetCondition(istioCR, operatorv1alpha1.NewReasonWithMessage(operatorv1alpha1.ConditionReasonIstioUninstallSucceeded))

		if err := removeInstallationFinalizer(ctx, i.Client, istioCR); err != nil {
			ctrl.Log.Error(err, "Error happened during istio installation finalizer removal")
			return described_errors.NewDescribedError(err, "Could not remove finalizer")
		}
	} else {
		statusHandler.SetCondition(istioCR, operatorv1alpha1.NewReasonWithMessage(operatorv1alpha1.ConditionReasonIstioInstallNotNeeded))
	}

	return nil
}

func hasInstallationFinalizer(istioCR *operatorv1alpha1.Istio) bool {
	return controllerutil.ContainsFinalizer(istioCR, installationFinalizer)
}

func addInstallationFinalizer(ctx context.Context, apiClient client.Client, istioCR *operatorv1alpha1.Istio) error {
	ctrl.Log.Info("Adding Istio installation finalizer")
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		finalizerCR := operatorv1alpha1.Istio{}
		if err := apiClient.Get(ctx, client.ObjectKeyFromObject(istioCR), &finalizerCR); err != nil {
			return err
		}
		if controllerutil.AddFinalizer(&finalizerCR, installationFinalizer) {
			if err := apiClient.Update(ctx, &finalizerCR); err != nil {
				return err
			}
		}
		istioCR.Finalizers = finalizerCR.Finalizers
		ctrl.Log.Info("Successfully added Istio installation finalizer")
		return nil
	})
}

func removeInstallationFinalizer(ctx context.Context, apiClient client.Client, istioCR *operatorv1alpha1.Istio) error {
	ctrl.Log.Info("Removing Istio installation finalizer")
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		finalizerCR := operatorv1alpha1.Istio{}
		if err := apiClient.Get(ctx, client.ObjectKeyFromObject(istioCR), &finalizerCR); err != nil {
			return err
		}
		if controllerutil.RemoveFinalizer(&finalizerCR, installationFinalizer) {
			if err := apiClient.Update(ctx, &finalizerCR); err != nil {
				return err
			}
		}
		istioCR.Finalizers = finalizerCR.Finalizers
		ctrl.Log.Info("Successfully removed Istio installation finalizer")
		return nil
	})
}
