package istio

import (
	"context"
	"fmt"

	"github.com/thoas/go-funk"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/resources"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/remove"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type uninstallArgs struct {
	k8sClient         client.Client
	istioCR           *operatorv1alpha2.Istio
	statusHandler     status.Status
	istioImageVersion istiooperator.IstioImageVersion
	istioClient       libraryClient
}

func uninstallIstio(ctx context.Context, args uninstallArgs) (istiooperator.IstioImageVersion, describederrors.DescribedError) {
	istioCR := args.istioCR
	istioImageVersion := args.istioImageVersion
	statusHandler := args.statusHandler
	istioClient := args.istioClient
	k8sClient := args.k8sClient

	ctrl.Log.Info("Starting Istio uninstall")

	istioResourceFinder, err := resources.NewIstioResourcesFinder(ctx, k8sClient, ctrl.Log)
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not read customer resources finder configuration")
	}

	clientResources, err := istioResourceFinder.FindUserCreatedIstioResources()
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not get customer resources from the cluster")
	}

	if len(clientResources) > 0 {
		funk.ForEach(clientResources, func(a resources.Resource) {
			ctrl.Log.Info("Customer resource is blocking Istio deletion", a.GVK.Kind, fmt.Sprintf("%s/%s", a.Namespace, a.Name))
		})
		statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonIstioCRsDangling))
		return istioImageVersion, describederrors.NewDescribedError(fmt.Errorf("could not delete Istio module instance since there are %d customer resources present", len(clientResources)),
			"There are Istio resources that block deletion. Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning").
			DisableErrorWrap().
			SetWarning().
			SetCondition(false)
	}

	err = istioClient.Uninstall(ctx)
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not uninstall istio")
	}

	warnings, err := remove.Sidecars(ctx, k8sClient, &ctrl.Log)
	if err != nil {
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not remove istio sidecars")
	}

	if len(warnings) > 0 {
		for _, w := range warnings {
			ctrl.Log.Info("Removing sidecar warning:", "name", w.Name, "namespace", w.Namespace, "kind", w.Kind, "message", w.Message)
		}
	}

	ctrl.Log.Info("Istio uninstall succeeded")
	statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonIstioUninstallSucceeded))

	if err := removeInstallationFinalizer(ctx, k8sClient, istioCR); err != nil {
		ctrl.Log.Error(err, "Error happened during istio installation finalizer removal")
		return istioImageVersion, describederrors.NewDescribedError(err, "Could not remove finalizer")
	}

	return istioImageVersion, nil
}
