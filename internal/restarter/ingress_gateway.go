package restarter

import (
	"context"
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	namespace      string = "istio-system"
	deploymentName string = "istio-ingressgateway"
)

type IngressGatewayRestarter struct {
	client        client.Client
	predicates    []filter.IngressGatewayPredicate
	statusHandler status.Status
}

func NewIngressGatewayRestarter(client client.Client, predicates []filter.IngressGatewayPredicate, statusHandler status.Status) *IngressGatewayRestarter {
	return &IngressGatewayRestarter{
		client:        client,
		predicates:    predicates,
		statusHandler: statusHandler,
	}
}

func (r *IngressGatewayRestarter) Restart(ctx context.Context, istioCR *v1alpha2.Istio) described_errors.DescribedError {
	ctrl.Log.Info("Restarting Istio Ingress Gateway")
	podList, err := getIngressGatewayPods(ctx, r.client)
	if err != nil {
		r.statusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonIngressGatewayRestartFailed))
		return described_errors.NewDescribedError(err, "Failed to get ingress gateway pods")
	}

	mustRestart := false

	for _, predicate := range r.predicates {
		evaluator, err := predicate.NewIngressGatewayEvaluator(ctx)
		if err != nil {
			r.statusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonIngressGatewayRestartFailed))
			return described_errors.NewDescribedError(err, "Cannot create evaluator")
		}
		for _, pod := range podList.Items {
			if evaluator.RequiresIngressGatewayRestart(pod) {
				mustRestart = true
				break
			}
		}

		if mustRestart {
			break
		}
	}

	if mustRestart {
		if err := RestartIngressGateway(ctx, r.client); err != nil {
			r.statusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonIngressGatewayRestartFailed))
			return described_errors.NewDescribedError(err, "Failed to restart ingress gateway")
		}
	}

	r.statusHandler.SetCondition(istioCR, v1alpha2.NewReasonWithMessage(v1alpha2.ConditionReasonIngressGatewayRestartSucceeded))
	ctrl.Log.Info("Successfully restarted Istio Ingress Gateway")
	return nil
}

func getIngressGatewayPods(ctx context.Context, k8sClient client.Client) (*v1.PodList, error) {

	ls := labels.SelectorFromSet(map[string]string{
		"app": "istio-ingressgateway",
	})

	list := v1.PodList{}
	err := k8sClient.List(ctx, &list, &client.ListOptions{Namespace: namespace, LabelSelector: ls})
	if err != nil {
		return nil, err
	}

	return &list, err

}

func RestartIngressGateway(ctx context.Context, k8sClient client.Client) error {
	ctrl.Log.Info("Restarting istio-ingressgateway")

	deployment := appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, &deployment)
	if err != nil {
		return err
	}

	patch := client.StrategicMergeFrom((&deployment).DeepCopy())
	deployment.Spec.Template.Annotations = annotations.AddRestartAnnotation(deployment.Spec.Template.Annotations)

	err = retry.RetryOnError(retry.DefaultRetry, func() error {
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
