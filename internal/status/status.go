package status

import (
	"context"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

func Update(ctx context.Context, client client.Client, istioCR *operatorv1alpha1.Istio, state operatorv1alpha1.State, condition metav1.Condition) (ctrl.Result, error) {
	istioCR.Status.State = state
	meta.SetStatusCondition(istioCR.Status.Conditions, condition)

	if err := client.Status().Update(ctx, istioCR); err != nil {
		ctrl.Log.Error(err, "Unable to update the status of Istio CR")
		return ctrl.Result{
			RequeueAfter: time.Minute * 5,
		}, err
	}

	return ctrl.Result{}, nil
}
