package istio

import (
	"context"
	"fmt"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	sidecarRemover "github.com/kyma-project/istio/operator/pkg/lib/sidecars/remove"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Installation struct {
	Client         LibraryClient
	IstioVersion   string
	IstioImageBase string
}

const (
	LastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
	installationFinalizer    string = "istios.operator.kyma-project.io/istio-installation"
)

// Reconcile runs Istio reconciliation to install, upgrade or uninstall Istio and returns the updated Istio CR.
func (i *Installation) Reconcile(ctx context.Context, client client.Client, istioCR operatorv1alpha1.Istio, defaultIstioOperatorPath, workingDir string) (operatorv1alpha1.Istio, error) {

	istioTag := fmt.Sprintf("%s-%s", i.IstioVersion, i.IstioImageBase)

	// We need to evaluate what changed since last reconciliation, to make sure we run Istio reconciliation only if it's necessary
	istioCRChanges, err := EvaluateIstioCRChanges(istioCR, istioTag)
	if err != nil {
		ctrl.Log.Error(err, "Error evaluating IstioCR changes")
		return istioCR, err
	}

	if !istioCRChanges.requireIstioDeletion() && !hasInstallationFinalizer(istioCR) {
		controllerutil.AddFinalizer(&istioCR, installationFinalizer)
		if err := client.Update(ctx, &istioCR); err != nil {
			return istioCR, err
		}
	}

	ctrl.Log.Info("Reconcile Istio installation", "Istio CR change evaluation", istioCRChanges)

	if istioCRChanges.requireInstall() {

		ctrl.Log.Info("Starting istio install", "istio version", i.IstioVersion, "istio image", i.IstioImageBase)

		// To have a better visibility of the manager state during install and upgrade, we update the status to Processing
		_, err = status.Update(ctx, client, &istioCR, operatorv1alpha1.Processing, metav1.Condition{})
		if err != nil {
			return istioCR, err
		}

		clusterConfiguration, err := clusterconfig.EvaluateClusterConfiguration(ctx, client)
		if err != nil {
			return istioCR, err
		}

		// As we define default IstioOperator values in a templated manifest, we need to apply the istio version and values from
		// Istio CR to this default configuration to get the final IstoOperator that is used for installing and updating Istio.
		mergedIstioOperatorPath, err := merge(&istioCR, defaultIstioOperatorPath, workingDir, TemplateData{IstioVersion: i.IstioVersion, IstioImageBase: i.IstioImageBase}, clusterConfiguration)
		if err != nil {
			return istioCR, err
		}

		err = i.Client.Install(mergedIstioOperatorPath)
		if err != nil {
			return istioCR, err
		}

		version, err := gatherer.GetIstioPodsVersion(ctx, client)
		if err != nil {
			return istioCR, err
		}

		if i.IstioVersion != version {
			return istioCR, fmt.Errorf("istio-system pods version: %s do not match target version: %s", version, i.IstioVersion)
		}

		ctrl.Log.Info("Istio install completed")

		// We use the installation finalizer to track if the deletion was already executed so can make the uninstall process more reliable.
	} else if istioCRChanges.requireIstioDeletion() && hasInstallationFinalizer(istioCR) {

		ctrl.Log.Info("Starting istio uninstall")

		_, err = status.Update(ctx, client, &istioCR, operatorv1alpha1.Deleting, metav1.Condition{})
		if err != nil {
			return istioCR, err
		}

		err := i.Client.Uninstall(ctx)
		if err != nil {
			return istioCR, err
		}

		warnings, err := sidecarRemover.RemoveSidecars(ctx, client, &ctrl.Log)
		if err != nil {
			return istioCR, err
		}

		if len(warnings) > 0 {
			for _, w := range warnings {
				ctrl.Log.Info("Removing sidecar warning:", "name", w.Name, "namespace", w.Namespace, "kind", w.Kind, "message", w.Message)
			}
		}

		controllerutil.RemoveFinalizer(&istioCR, installationFinalizer)
		if err := client.Update(ctx, &istioCR); err != nil {
			ctrl.Log.Error(err, "Error happened during istio installation finalizer removal")
			return istioCR, err
		}

		ctrl.Log.Info("Istio uninstall completed")
	}

	return istioCR, nil
}

func hasInstallationFinalizer(istioCR operatorv1alpha1.Istio) bool {
	return controllerutil.ContainsFinalizer(&istioCR, installationFinalizer)

}
