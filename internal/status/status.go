package status

import (
	"context"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Status interface {
	UpdateToProcessing(ctx context.Context, description string, client client.Client, istioCR *operatorv1alpha1.Istio) error
	UpdateToError(ctx context.Context, err described_errors.DescribedError, client client.Client, istioCR *operatorv1alpha1.Istio) error
	UpdateToDeleting(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio) error
	UpdateToReady(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio) error
}

func NewDefaultStatusHandler() DefaultStatusHandler {
	return DefaultStatusHandler{}
}

type DefaultStatusHandler struct{}

func update(ctx context.Context, apiClient client.Client, istioCR *operatorv1alpha1.Istio) error {

	newStatus := istioCR.Status

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if getErr := apiClient.Get(ctx, client.ObjectKeyFromObject(istioCR), istioCR); getErr != nil {
			return getErr
		}

		istioCR.Status = newStatus

		if updateErr := apiClient.Status().Update(ctx, istioCR); updateErr != nil {
			return updateErr
		}

		return nil
	})
}

func (DefaultStatusHandler) UpdateToProcessing(ctx context.Context, description string, client client.Client, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Processing
	istioCR.Status.Description = description
	return update(ctx, client, istioCR)
}

func (DefaultStatusHandler) UpdateToError(ctx context.Context, err described_errors.DescribedError, client client.Client, istioCR *operatorv1alpha1.Istio) error {
	if err.Level() == described_errors.Warning {
		istioCR.Status.State = operatorv1alpha1.Warning
	} else {
		istioCR.Status.State = operatorv1alpha1.Error
	}
	istioCR.Status.Description = err.Description()
	return update(ctx, client, istioCR)
}

func (DefaultStatusHandler) UpdateToDeleting(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Deleting
	istioCR.Status.Description = "Removing Istio resources"
	return update(ctx, client, istioCR)
}

func (DefaultStatusHandler) UpdateToReady(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Ready
	istioCR.Status.Description = ""
	return update(ctx, client, istioCR)
}
