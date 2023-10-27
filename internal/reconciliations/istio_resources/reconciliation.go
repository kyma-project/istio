package istio_resources

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type ResourcesReconciliation interface {
	Reconcile(ctx context.Context, istioCR v1alpha1.Istio) described_errors.DescribedError
}

type ResourcesReconciler struct {
	client         client.Client
	templateValues map[string]string
}

func NewReconciler(client client.Client) *ResourcesReconciler {
	return &ResourcesReconciler{
		client: client,
	}
}

type Resource interface {
	Name() string
	apply(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, templateValues map[string]string) (controllerutil.OperationResult, error)
}

func (r *ResourcesReconciler) Reconcile(ctx context.Context, istioCR v1alpha1.Istio) described_errors.DescribedError {
	ctrl.Log.Info("Reconciling istio resources")

	// We get the istio resources in the reconciliation instead of the initialisation of the reconciler, because we need to fetch information from the cluster using the kube client.
	// During the creation of the reconciler the manager is not initialised yet and therefore the kube client cannot be used, yet.
	resources, err := getResources(ctx, r.client)
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

// getResources returns all Istio resources required for the reconciliation specific for the given hyperscaler.
func getResources(ctx context.Context, k8sClient client.Client) ([]Resource, error) {

	istioResources := []Resource{NewEnvoyFilterAllowPartialReferer(k8sClient)}
	istioResources = append(istioResources, NewGatewayKyma(k8sClient))
	istioResources = append(istioResources, NewVirtualServiceHealthz(k8sClient))
	istioResources = append(istioResources, NewPeerAuthenticationMtls(k8sClient))
	istioResources = append(istioResources, NewConfigMapControlPlane(k8sClient))
	istioResources = append(istioResources, NewConfigMapMesh(k8sClient))
	istioResources = append(istioResources, NewConfigMapPerformance(k8sClient))
	istioResources = append(istioResources, NewConfigMapService(k8sClient))
	istioResources = append(istioResources, NewConfigMapWorkload(k8sClient))

	isAws, err := clusterconfig.IsHyperscalerAWS(ctx, k8sClient)
	if err != nil {
		return nil, err
	}

	if isAws {
		istioResources = append(istioResources, NewProxyProtocolEnvoyFilter(k8sClient))
	}

	return istioResources, nil
}
