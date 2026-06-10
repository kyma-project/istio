package istioresources

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/api/v1alpha2"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/kyma-project/istio/operator/internal/clusterconfig/strategy"
	"github.com/kyma-project/istio/operator/internal/describederrors"
	"github.com/kyma-project/istio/operator/internal/istiofeatures"
)

type ResourcesReconciliation interface {
	Reconcile(ctx context.Context, istioCR v1alpha2.Istio, clusterStrategy *strategy.Hyperscaler) describederrors.DescribedError
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
	reconcile(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, templateValues map[string]string) (controllerutil.OperationResult, error)
}

func (r *ResourcesReconciler) Reconcile(ctx context.Context, istioCR v1alpha2.Istio, clusterStrategy *strategy.Hyperscaler) describederrors.DescribedError {
	ctrl.Log.Info("Reconciling Istio resources")

	features, featErr := istiofeatures.Get(ctx, r.client)
	if featErr != nil {
		ctrl.Log.V(1).Info("Could not get Istio features for resource reconciliation, proceeding with defaults", "error", featErr)
	}

	resources := getResources(clusterStrategy, istioCR, features)

	owner := metav1.OwnerReference{
		APIVersion: istioCR.APIVersion,
		Kind:       istioCR.Kind,
		Name:       istioCR.Name,
		UID:        istioCR.UID,
	}

	for _, resource := range resources {
		ctrl.Log.Info("Reconciling Istio resource", "name", resource.Name())
		result, reconcileErr := resource.reconcile(ctx, r.client, owner, r.templateValues)

		if reconcileErr != nil {
			return describederrors.NewDescribedError(reconcileErr, fmt.Sprintf("Could not reconcile Istio resource %s", resource.Name()))
		}
		ctrl.Log.Info("Reconciled Istio resource", "name", resource.Name(), "result", result)
	}

	ctrl.Log.Info("Successfully reconciled Istio resources")

	return nil
}

// getResources returns all Istio resources required for the reconciliation specific for the given hyperscaler strategy.
func getResources(clusterStrategy *strategy.Hyperscaler, istioCR v1alpha2.Istio, features istiofeatures.IstioFeatures) []Resource {
	// @Ressetkk: this logic needs to be moved to main reconciliation loop.
	// Remove dynamic assignment of resource reconcilers.
	// Can't write proper tests if I don't know which resources are reconciled in the loop.
	if istioCR.DeletionTimestamp != nil && !istioCR.DeletionTimestamp.IsZero() {
		// NewPeerAuthenticationMtls does not delete resources
		// NewProxyProtocolEnvoyFilter fails because CRDs are removed before it can delete the EnvoyFilter
		return []Resource{
			NewNetworkPolicies(true),
			NewVPA(true),
			NewControlPlaneVPA(true),
		}
	}
	istioResources := []Resource{
		NewPeerAuthenticationMtls(false),
		NewNetworkPolicies(!istioCR.Spec.NetworkPoliciesEnabled),
		NewVPA(false),
		NewControlPlaneVPA(!features.EnableControlPlaneVPA),
	}

	if clusterStrategy != nil && clusterStrategy.LB != nil {
		istioResources = append(istioResources, NewProxyProtocolEnvoyFilter(!clusterStrategy.LB.RequiresProxyProtocolEnvoyFilter()))
	}

	return istioResources
}
