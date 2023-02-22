package istio

import (
	"context"
	"fmt"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Installation struct {
	Client         LibraryClient
	IstioVersion   string
	IstioImageBase string
}

const (
	LastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
)

// Reconcile runs Istio reconciliation to install, upgrade or uninstall Istio.
func (i *Installation) Reconcile(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio, defaultIstioOperatorPath, workingDir string) error {

	istioTag := fmt.Sprintf("%s-%s", i.IstioVersion, i.IstioImageBase)

	// We need to evaluate what changed since last reconciliation, to make sure we run Istio reconciliation only if it's necessary
	istioCRChanges, err := EvaluateIstioCRChanges(istioCR, istioTag)
	if err != nil {
		ctrl.Log.Error(err, "Error evaluating IstioCR changes")
		return err
	}

	if !istioCRChanges.MustBeReconciled() {
		ctrl.Log.Info("Reconciliation of Istio installation was skipped")
		return nil
	}

	ctrl.Log.Info("Reconcile Istio installation")

	if istioCRChanges.requireInstall() {

		// To have a better visibility of the manager state during install and update, we update the status to Processing
		_, err = status.Update(ctx, client, istioCR, operatorv1alpha1.Processing, metav1.Condition{})
		if err != nil {
			return err
		}

		ctrl.Log.Info("Starting istio install", "istio version", i.IstioVersion, "istio image", i.IstioImageBase)

		// As we define default IstioOperator values in a templated manifest, we need to apply the istio version and values from
		// Istio CR to this default configuration to get the final IstoOperator that is used for installing and updating Istio.
		mergedIstioOperatorPath, err := merge(istioCR, defaultIstioOperatorPath, workingDir, TemplateData{IstioVersion: i.IstioVersion, IstioImageBase: i.IstioImageBase})
		if err != nil {
			return err
		}

		err = i.Client.Install(mergedIstioOperatorPath)
		if err != nil {
			return err
		}
		ctrl.Log.Info("Istio install completed")
	}

	if istioCRChanges.requireIstioDeletion() {
		ctrl.Log.Info("Starting istio uninstall")
		err := i.Client.Uninstall(ctx)
		if err != nil {
			return err
		}
		ctrl.Log.Info("Istio uninstall completed")
	}

	return nil
}
