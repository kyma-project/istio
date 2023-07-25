package ingress_gateway

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
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

type Reconciliation interface {
	Reconcile(ctx context.Context) described_errors.DescribedError
}

type Reconciler struct {
	Client     client.Client
	Predicates []filter.IngressGatewayPredicate
}

func (r Reconciler) Reconcile(ctx context.Context) described_errors.DescribedError {
	ctrl.Log.Info("Reconciling Istio ingress gateway")

	podList, err := getIngressGatewayPods(ctx, r.Client)
	if err != nil {
		return described_errors.NewDescribedError(err, "Failed to get ingress gateway pods")
	}

	mustRestart := false

	for _, pod := range podList.Items {
		for _, predicate := range r.Predicates {
			shouldRestart, err := predicate.RequiresIngressGatewayRestart(ctx, pod)
			if err != nil {
				return described_errors.NewDescribedError(err, "Cannot check predicate")
			}
			if shouldRestart {
				mustRestart = true
				break
			}
		}

		if mustRestart {
			break
		}
	}

	if mustRestart {
		// TODO: just re-used the existing logic, we can also think about restarting the pod directly
		if err := restartIngressGateway(ctx, r.Client); err != nil {
			return described_errors.NewDescribedError(err, "Failed to restart ingress gateway")
		}
	}

	ctrl.Log.Info("Successfully reconciled Istio ingress gateway")
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

// TODO this code is duplicated in istio installation package, we should reuse it
func restartIngressGateway(ctx context.Context, k8sClient client.Client) error {
	ctrl.Log.Info("Restarting istio-ingressgateway")

	deployment := appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, &deployment)
	if err != nil {
		return err
	}
	deployment.Spec.Template.Annotations = annotations.AddRestartAnnotation(deployment.Spec.Template.Annotations)
	err = k8sClient.Update(ctx, &deployment)
	if err != nil {
		return err
	}
	ctrl.Log.Info("istio-ingressgateway restarted")

	return nil
}
