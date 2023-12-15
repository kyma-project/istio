package controllers

import (
	"context"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type status interface {
	updateToProcessing(ctx context.Context, description string, istioCR *operatorv1alpha1.Istio) error
	updateToError(ctx context.Context, err described_errors.DescribedError, istioCR *operatorv1alpha1.Istio) error
	updateToDeleting(ctx context.Context, istioCR *operatorv1alpha1.Istio) error
	updateToReady(ctx context.Context, istioCR *operatorv1alpha1.Istio) error
}

func newStatusHandler(client client.Client) StatusHandler {
	return StatusHandler{
		client: client,
	}
}

type StatusHandler struct {
	client client.Client
}

func (d StatusHandler) update(ctx context.Context, istioCR *operatorv1alpha1.Istio) error {
	newStatus := istioCR.Status

	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if getErr := d.client.Get(ctx, client.ObjectKeyFromObject(istioCR), istioCR); getErr != nil {
			return getErr
		}

		istioCR.Status = newStatus

		if updateErr := d.client.Status().Update(ctx, istioCR); updateErr != nil {
			return updateErr
		}

		return nil
	})
}

func (d StatusHandler) updateToProcessing(ctx context.Context, description string, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Processing
	istioCR.Status.Description = description
	return d.update(ctx, istioCR)
}

func (d StatusHandler) updateToError(ctx context.Context, err described_errors.DescribedError, istioCR *operatorv1alpha1.Istio) error {
	if err.Level() == described_errors.Warning {
		istioCR.Status.State = operatorv1alpha1.Warning
	} else {
		istioCR.Status.State = operatorv1alpha1.Error
	}
	istioCR.Status.Description = err.Description()
	return d.update(ctx, istioCR)
}

func (d StatusHandler) updateToDeleting(ctx context.Context, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Deleting
	istioCR.Status.Description = "Removing Istio resources"
	return d.update(ctx, istioCR)
}

func (d StatusHandler) updateToReady(ctx context.Context, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Ready
	istioCR.Status.Description = ""
	return d.update(ctx, istioCR)
}
