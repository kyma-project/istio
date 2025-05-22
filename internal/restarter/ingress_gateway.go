package restarter

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	utilretry "k8s.io/client-go/util/retry"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	ingressNamespace      string = "istio-system"
	ingressDeploymentName string = "istio-ingressgateway"
)

type IngressGatewayRestarter struct {
	client        client.Client
	predicates    []predicates.IngressGatewayPredicate
	statusHandler status.Status
}

func NewIngressGatewayRestarter(client client.Client, predicates []predicates.IngressGatewayPredicate, statusHandler status.Status) *IngressGatewayRestarter {
	return &IngressGatewayRestarter{
		client:        client,
		predicates:    predicates,
		statusHandler: statusHandler,
	}
}

func (r *IngressGatewayRestarter) Restart(ctx context.Context, istioCR *v1alpha2.Istio) (describederrors.DescribedError, bool) {
	ctrl.Log.Info("Restarting Istio Ingress Gateway")

	r.predicates = append(r.predicates, predicates.NewIngressGatewayRestartPredicate(istioCR))
	for _, predicate := range r.predicates {
		evaluator, err := predicate.NewIngressGatewayEvaluator(ctx)
		if err != nil {
			return describederrors.NewDescribedError(err, "Could not create Ingress Gateway restart evaluator"), false
		}

		if evaluator.RequiresIngressGatewayRestart() {
			err = restartIngressGateway(ctx, r.client)
			if err != nil {
				r.statusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonIngressGatewayRestartFailed))
				return describederrors.NewDescribedError(err, "Failed to restart Ingress Gateway"), false
			}
		}
	}

	r.statusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonIngressGatewayRestartSucceeded))
	ctrl.Log.Info("Successfully restarted Istio Ingress Gateway")
	return nil, false
}

func restartIngressGateway(ctx context.Context, k8sClient client.Client) error {
	ctrl.Log.Info("Restarting istio-ingressgateway")

	deployment := appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: ingressNamespace, Name: ingressDeploymentName}, &deployment)
	if err != nil {
		// If ingress gateway deployment is missing, we should not fail, as it may have not yet been created
		// In that case, the upcoming creation of the deployment will do the same thing as we would require from the restart
		if k8sErrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	patch := client.StrategicMergeFrom((&deployment).DeepCopy())
	deployment.Spec.Template.Annotations = annotations.AddRestartAnnotation(deployment.Spec.Template.Annotations)

	err = retry.OnError(utilretry.DefaultRetry, func() error {
		err = k8sClient.Patch(ctx, &deployment, patch)
		if err != nil {
			ctrl.Log.Info("Retrying ingress gateway restart")
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	ctrl.Log.Info("istio-ingressgateway restarted")

	return nil
}
