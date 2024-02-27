package istio_resources

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/api/v1alpha2"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ResourcesReconciliation interface {
	Reconcile(ctx context.Context, istioCR v1alpha2.Istio) described_errors.DescribedError
}

type ResourcesReconciler struct {
	client         client.Client
	hsClient       clusterconfig.Hyperscaler
	templateValues map[string]string
}

func NewReconciler(client client.Client, hsClient clusterconfig.Hyperscaler) *ResourcesReconciler {
	return &ResourcesReconciler{
		client:   client,
		hsClient: hsClient,
	}
}

type Resource interface {
	Name() string
	reconcile(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, templateValues map[string]string) (controllerutil.OperationResult, error)
}

func (r *ResourcesReconciler) Reconcile(ctx context.Context, istioCR v1alpha2.Istio) described_errors.DescribedError {
	ctrl.Log.Info("Reconciling Istio resources")

	resources, err := getResources(r.client, r.hsClient)
	if err != nil {
		ctrl.Log.Error(err, "Failed to initialise Istio resources")
		return described_errors.NewDescribedError(err, "Istio controller failed to initialise Istio resources")
	}

	err = r.getTemplateValues(ctx, istioCR)
	if err != nil {
		return described_errors.NewDescribedError(err, "Could not get template values for istio resources")
	}

	owner := metav1.OwnerReference{
		APIVersion: istioCR.APIVersion,
		Kind:       istioCR.Kind,
		Name:       istioCR.Name,
		UID:        istioCR.UID,
	}

	for _, resource := range resources {
		ctrl.Log.Info("Reconciling Istio resource", "name", resource.Name())
		result, err := resource.reconcile(ctx, r.client, owner, r.templateValues)

		if err != nil {
			return described_errors.NewDescribedError(err, fmt.Sprintf("Could not reconcile Istio resource %s", resource.Name()))
		}
		ctrl.Log.Info("Reconciled Istio resource", "name", resource.Name(), "result", result)
	}

	ctrl.Log.Info("Successfully reconciled Istio resources")

	return nil
}

func (r *ResourcesReconciler) getTemplateValues(ctx context.Context, istioCR v1alpha2.Istio) error {
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

// getResources returns all Istio resources required for the reconciliation specific for the given hyperscaler.
func getResources(k8sClient client.Client, hsClient clusterconfig.Hyperscaler) ([]Resource, error) {
	istioResources := []Resource{NewEnvoyFilterAllowPartialReferer(k8sClient)}
	istioResources = append(istioResources, NewPeerAuthenticationMtls(k8sClient))

	isAws := hsClient.IsAws()
	if isAws {
		istioResources = append(istioResources, NewProxyProtocolEnvoyFilter(k8sClient))
	}

	return istioResources, nil
}
