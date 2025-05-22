package status

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
)

type Status interface {
	UpdateToProcessing(ctx context.Context, istioCR *operatorv1alpha2.Istio) error
	UpdateToDeleting(ctx context.Context, istioCR *operatorv1alpha2.Istio) error
	UpdateToReady(ctx context.Context, istioCR *operatorv1alpha2.Istio) error
	UpdateToError(ctx context.Context, istioCR *operatorv1alpha2.Istio, err describederrors.DescribedError) error
	SetCondition(istioCR *operatorv1alpha2.Istio, reason operatorv1alpha2.ReasonWithMessage)
}

type Handler struct {
	client client.Client
}

func NewStatusHandler(client client.Client) Status {
	return Handler{
		client: client,
	}
}

func (d Handler) update(ctx context.Context, istioCR *operatorv1alpha2.Istio) error {
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

func (d Handler) UpdateToProcessing(ctx context.Context, istioCR *operatorv1alpha2.Istio) error {
	istioCR.Status.State = operatorv1alpha2.Processing
	istioCR.Status.Description = "Reconciling Istio"
	return d.update(ctx, istioCR)
}

func (d Handler) UpdateToDeleting(ctx context.Context, istioCR *operatorv1alpha2.Istio) error {
	istioCR.Status.State = operatorv1alpha2.Deleting
	istioCR.Status.Description = "Deleting Istio"
	return d.update(ctx, istioCR)
}

func (d Handler) UpdateToReady(ctx context.Context, istioCR *operatorv1alpha2.Istio) error {
	istioCR.Status.State = operatorv1alpha2.Ready
	istioCR.Status.Description = ""
	return d.update(ctx, istioCR)
}

func (d Handler) UpdateToError(ctx context.Context, istioCR *operatorv1alpha2.Istio, err describederrors.DescribedError) error {
	if err.Level() == describederrors.Warning {
		istioCR.Status.State = operatorv1alpha2.Warning
	} else {
		istioCR.Status.State = operatorv1alpha2.Error
	}
	istioCR.Status.Description = err.Description()
	return d.update(ctx, istioCR)
}

func (d Handler) SetCondition(istioCR *operatorv1alpha2.Istio, reason operatorv1alpha2.ReasonWithMessage) {
	if istioCR.Status.Conditions == nil {
		istioCR.Status.Conditions = &[]metav1.Condition{}
	}
	condition := operatorv1alpha2.ConditionFromReason(reason)
	if condition != nil {
		meta.SetStatusCondition(istioCR.Status.Conditions, *condition)
	} else {
		ctrl.Log.Error(errors.New("condition not found"), "Unable to find condition from reason", "reason", reason)
	}
}
