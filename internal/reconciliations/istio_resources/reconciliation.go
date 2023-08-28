package istio_resources

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
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
	client         client.Client
	resources      []Resource
	templateValues map[string]string
}

func NewReconciler(client client.Client, resources []Resource) Reconciler {
	return Reconciler{
		client:    client,
		resources: resources,
	}
}

type Resource interface {
	Name() string
	apply(ctx context.Context, k8sClient client.Client, templateValues map[string]string) (controllerutil.OperationResult, error)
}

func (r Reconciler) Reconcile(ctx context.Context) described_errors.DescribedError {
	ctrl.Log.Info("Reconciling istio resources")

	err := r.initTemplateValues(ctx)
	if err != nil {
		return described_errors.NewDescribedError(err, "Could not initialize template values for istio resources")
	}

	for _, resource := range r.resources {
		ctrl.Log.Info("Reconciling istio resource", "name", resource.Name())
		result, err := resource.apply(ctx, r.client, r.templateValues)

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

func (r Reconciler) initTemplateValues(ctx context.Context) error {
	if len(r.templateValues) == 0 {
		r.templateValues = make(map[string]string)
	}

	domainName := clusterconfig.LocalKymaDomain
	flavour, err := clusterconfig.DiscoverClusterFlavour(ctx, r.client)
	if err != nil {
		return err
	}
	if flavour == clusterconfig.Gardener {
		domainName, err = clusterconfig.GetDomainName(ctx, r.client)
		if err != nil {
			return err
		}
	}
	r.templateValues["DomainName"] = domainName

	return nil
}
