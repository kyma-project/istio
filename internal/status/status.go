package status

import (
	"context"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type Status interface {
	SetProcessing(ctx context.Context, description string, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error)
	SetReady(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error)
	SetError(ctx context.Context, err described_errors.DescribedError, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error)
	SetDeleting(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error)
}

func NewDefaultStatusHandler() DefaultStatusHandler {
	return DefaultStatusHandler{}
}

type DefaultStatusHandler struct{}

func update(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error) {
	meta.SetStatusCondition(istioCR.Status.Conditions, condition)

	if err := client.Status().Update(ctx, istioCR); err != nil {
		ctrl.Log.Error(err, "Unable to update the status of Istio CR")
		return ctrl.Result{
			RequeueAfter: time.Minute * 5,
		}, err
	}
	if len(retryTime) > 0 {
		return ctrl.Result{RequeueAfter: retryTime[0]}, nil
	}
	return ctrl.Result{}, nil
}

func (DefaultStatusHandler) SetReady(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error) {
	istioCR.Status.State = operatorv1alpha1.Ready
	return update(ctx, client, istioCR, condition, retryTime...)
}

func (DefaultStatusHandler) SetProcessing(ctx context.Context, description string, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error) {
	istioCR.Status.State = operatorv1alpha1.Processing
	istioCR.Status.Description = description
	return update(ctx, client, istioCR, condition, retryTime...)
}

func (DefaultStatusHandler) SetError(ctx context.Context, err described_errors.DescribedError, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error) {
	istioCR.Status.State = operatorv1alpha1.Error
	istioCR.Status.Description = err.Description()
	return update(ctx, client, istioCR, condition, retryTime...)
}

func (DefaultStatusHandler) SetDeleting(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio, condition metav1.Condition, retryTime ...time.Duration) (ctrl.Result, error) {
	istioCR.Status.State = operatorv1alpha1.Deleting
	return update(ctx, client, istioCR, condition, retryTime...)
}
