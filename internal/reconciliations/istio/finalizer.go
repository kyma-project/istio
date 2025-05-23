package istio

import (
	"context"

	"k8s.io/client-go/util/retry"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
)

const (
	installationFinalizer string = "istios.operator.kyma-project.io/istio-installation"
)

func hasInstallationFinalizer(istioCR *operatorv1alpha2.Istio) bool {
	return controllerutil.ContainsFinalizer(istioCR, installationFinalizer)
}

func addInstallationFinalizer(ctx context.Context, apiClient client.Client, istioCR *operatorv1alpha2.Istio) error {
	ctrl.Log.Info("Adding Istio installation finalizer")
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		finalizerCR := operatorv1alpha2.Istio{}
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

func removeInstallationFinalizer(ctx context.Context, apiClient client.Client, istioCR *operatorv1alpha2.Istio) error {
	ctrl.Log.Info("Removing Istio installation finalizer")
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		finalizerCR := operatorv1alpha2.Istio{}
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
