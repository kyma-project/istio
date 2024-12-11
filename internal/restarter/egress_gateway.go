package restarter

import (
	"context"
	"github.com/kyma-project/istio/operator/pkg/lib/egressgateway"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	egressNamespace      string = "istio-system"
	egressDeploymentName string = "istio-egressgateway"
)

type EgressGatewayRestarter struct {
	client        client.Client
	predicates    []filter.EgressGatewayPredicate
	statusHandler status.Status
}

func NewEgressGatewayRestarter(client client.Client, predicates []filter.EgressGatewayPredicate, statusHandler status.Status) *EgressGatewayRestarter {
	return &EgressGatewayRestarter{
		client:        client,
		predicates:    predicates,
		statusHandler: statusHandler,
	}
}

func (r *EgressGatewayRestarter) Restart(ctx context.Context, istioCR *v1alpha2.Istio) (described_errors.DescribedError, bool) {
	ctrl.Log.Info("Restarting Istio Egress Gateway")

	r.predicates = append(r.predicates, egressgateway.NewRestartPredicate(istioCR))
	for _, predicate := range r.predicates {
		evaluator, err := predicate.NewEgressGatewayEvaluator(ctx)
		if err != nil {
			return described_errors.NewDescribedError(err, "Could not create Egress Gateway restart evaluator"), false
		}

		if evaluator.RequiresEgressGatewayRestart() {
			err = restartEgressGateway(ctx, r.client)
			if err != nil {
				r.statusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonEgressGatewayRestartFailed))
				return described_errors.NewDescribedError(err, "Failed to restart Engress Gateway"), false
			}
		}
	}

	r.statusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonEgressGatewayRestartSucceeded))
	ctrl.Log.Info("Successfully restarted Istio Egress Gateway")
	return nil, false
}

func restartEgressGateway(ctx context.Context, k8sClient client.Client) error {
	ctrl.Log.Info("Restarting istio-egressgateway")

	deployment := appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: egressNamespace, Name: egressDeploymentName}, &deployment)
	if err != nil {
		// If egress gateway deployment is missing, we should not fail, as it may have not yet been created
		// In that case, the upcoming creation of the deployment will do the same thing as we would require from the restart
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	patch := client.StrategicMergeFrom((&deployment).DeepCopy())
	deployment.Spec.Template.Annotations = annotations.AddRestartAnnotation(deployment.Spec.Template.Annotations)

	err = retry.RetryOnError(retry.DefaultRetry, func() error {
		err = k8sClient.Patch(ctx, &deployment, patch)
		if err != nil {
			ctrl.Log.Info("Retrying egress gateway restart")
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	ctrl.Log.Info("istio-egressgateway restarted")

	return nil
}
