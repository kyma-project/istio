package istio

import (
	"context"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/status"
)

type InstallationReconciliation interface {
	Reconcile(ctx context.Context, istioCR *operatorv1alpha2.Istio, statusHandler status.Status) (istiooperator.IstioImageVersion, describederrors.DescribedError)
}

type Installation struct {
	IstioClient libraryClient
	Client      client.Client
	Merger      istiooperator.Merger
}

// Reconcile runs Istio reconciliation to install, upgrade or uninstall Istio and returns the updated Istio CR.
func (i *Installation) Reconcile(
	ctx context.Context,
	istioCR *operatorv1alpha2.Istio,
	statusHandler status.Status,
) (istiooperator.IstioImageVersion, describederrors.DescribedError) {
	istioImageVersion, err := i.Merger.GetIstioImageVersion()
	if err != nil {
		ctrl.Log.Error(err, "Error getting Istio version from istio operator file")
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not get Istio version from istio operator file")
	}

	if istioCR.DeletionTimestamp.IsZero() {
		args := installArgs{
			client:              i.Client,
			istioCR:             istioCR,
			statusHandler:       statusHandler,
			istioOperatorMerger: i.Merger,
			istioImageVersion:   istioImageVersion,
			istioClient:         i.IstioClient,
		}
		return installIstio(ctx, args)
	}

	// We use the installation finalizer to track if the deletion was already executed so can make the uninstallation process more reliable.
	if !istioCR.DeletionTimestamp.IsZero() && hasInstallationFinalizer(istioCR) {
		args := uninstallArgs{
			k8sClient:         i.Client,
			istioCR:           istioCR,
			statusHandler:     statusHandler,
			istioImageVersion: istioImageVersion,
			istioClient:       i.IstioClient,
		}
		return uninstallIstio(ctx, args)
	}

	statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonIstioInstallNotNeeded))

	return istioImageVersion, nil
}
