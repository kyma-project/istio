package istio_resources

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ResourcesReconciliation interface {
	Reconcile(ctx context.Context, istioCR v1alpha1.Istio) described_errors.DescribedError
}

type ResourcesReconciler struct {
	client         client.Client
	resources      []Resource
	templateValues map[string]string
}

func NewReconciler(client client.Client, resources []Resource) *ResourcesReconciler {
	return &ResourcesReconciler{
		client:    client,
		resources: resources,
	}
}

type Resource interface {
	Name() string
	apply(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, templateValues map[string]string) (controllerutil.OperationResult, error)
}

func (r *ResourcesReconciler) Reconcile(ctx context.Context, istioCR v1alpha1.Istio) described_errors.DescribedError {
	ctrl.Log.Info("Reconciling istio resources")

	err := r.getTemplateValues(ctx, istioCR)
	if err != nil {
		return described_errors.NewDescribedError(err, "Could not get template values for istio resources")
	}

	owner := metav1.OwnerReference{
		APIVersion: istioCR.APIVersion,
		Kind:       istioCR.Kind,
		Name:       istioCR.Name,
		UID:        istioCR.UID,
	}

	for _, resource := range r.resources {
		ctrl.Log.Info("Reconciling istio resource", "name", resource.Name())
		result, err := resource.apply(ctx, r.client, owner, r.templateValues)

		if err != nil {
			return described_errors.NewDescribedError(err, fmt.Sprintf("Could not reconcile istio resource %s", resource.Name()))
		}
		ctrl.Log.Info("Reconciled istio resource", "name", resource.Name(), "result", result)
	}

	ctrl.Log.Info("Successfully reconciled istio resources")

	return nil
}

func (r *ResourcesReconciler) getTemplateValues(ctx context.Context, istioCR v1alpha1.Istio) error {
	if len(r.templateValues) == 0 {
		r.templateValues = make(map[string]string)
	}
	_, found := r.templateValues["DomainName"]
	if !found {
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
	}

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
