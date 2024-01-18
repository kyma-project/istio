package status

import (
	"context"
	"errors"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Status interface {
	UpdateToProcessing(ctx context.Context, istioCR *operatorv1alpha1.Istio) error
	UpdateToDeleting(ctx context.Context, istioCR *operatorv1alpha1.Istio) error
	UpdateToReady(ctx context.Context, istioCR *operatorv1alpha1.Istio) error
	UpdateToError(ctx context.Context, istioCR *operatorv1alpha1.Istio, err described_errors.DescribedError) error
	SetCondition(istioCR *operatorv1alpha1.Istio, reason operatorv1alpha1.ReasonWithMessage)
}

func NewStatusHandler(client client.Client) StatusHandler {
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

func (d StatusHandler) UpdateToProcessing(ctx context.Context, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Processing
	istioCR.Status.Description = "Reconciling Istio"
	return d.update(ctx, istioCR)
}

func (d StatusHandler) UpdateToDeleting(ctx context.Context, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Deleting
	istioCR.Status.Description = "Deleting Istio"
	return d.update(ctx, istioCR)
}

func (d StatusHandler) UpdateToReady(ctx context.Context, istioCR *operatorv1alpha1.Istio) error {
	istioCR.Status.State = operatorv1alpha1.Ready
	istioCR.Status.Description = ""
	return d.update(ctx, istioCR)
}

func (d StatusHandler) UpdateToError(ctx context.Context, istioCR *operatorv1alpha1.Istio, err described_errors.DescribedError) error {
	if err.Level() == described_errors.Warning {
		istioCR.Status.State = operatorv1alpha1.Warning
	} else {
		istioCR.Status.State = operatorv1alpha1.Error
	}
	istioCR.Status.Description = err.Description()
	return d.update(ctx, istioCR)
}

func (d StatusHandler) SetCondition(istioCR *operatorv1alpha1.Istio, reason operatorv1alpha1.ReasonWithMessage) {
	if istioCR.Status.Conditions == nil {
		istioCR.Status.Conditions = &[]metav1.Condition{}
	}
	condition := operatorv1alpha1.ConditionFromReason(reason)
	if condition != nil {
		meta.SetStatusCondition(istioCR.Status.Conditions, *condition)
	} else {
		ctrl.Log.Error(errors.New("condition not found"), "Unable to find condition from reason", "reason", reason)
	}
}
