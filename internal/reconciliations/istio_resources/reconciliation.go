package istio_resources

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type Reconciliation interface {
	Reconcile(ctx context.Context) described_errors.DescribedError
}

type Reconciler struct {
	client    client.Client
	resources []Resource
}

func NewReconciler(client client.Client, resources []Resource) Reconciler {
	return Reconciler{
		client:    client,
		resources: resources,
	}
}

type Resource interface {
	Name() string
	apply(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error)
}

func (r Reconciler) Reconcile(ctx context.Context) described_errors.DescribedError {
	ctrl.Log.Info("Reconciling istio resources")

	for _, resource := range r.resources {
		ctrl.Log.Info("Reconciling istio resource", "name", resource.Name())
		result, err := resource.apply(ctx, r.client)

		if err != nil {
			return described_errors.NewDescribedError(err, fmt.Sprintf("Could not reconcile istio resource %s", resource.Name()))
		}
		ctrl.Log.Info("Reconciled istio resource", "name", resource.Name(), "result", result)
	}

	ctrl.Log.Info("Successfully reconciled istio resources")

	return nil
}

func annotateWithDisclaimer(ctx context.Context, resource unstructured.Unstructured, k8sClient client.Client) error {
	annotations := resource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[istio.DisclaimerKey] = istio.DisclaimerValue
	resource.SetAnnotations(annotations)

	err := k8sClient.Update(ctx, &resource)
	return err
}
