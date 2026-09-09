package istioresources

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/kyma-project/istio/operator/internal/resources"
	"github.com/kyma-project/istio/operator/pkg/labels"

	ctrl "sigs.k8s.io/controller-runtime"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed gateway_api_crds/gateway.networking.k8s.io_backendtlspolicies.yaml
var gatewayAPIBackendTLSPoliciesCRD []byte

//go:embed gateway_api_crds/gateway.networking.k8s.io_gatewayclasses.yaml
var gatewayAPIGatewayClassesCRD []byte

//go:embed gateway_api_crds/gateway.networking.k8s.io_gateways.yaml
var gatewayAPIGatewaysCRD []byte

//go:embed gateway_api_crds/gateway.networking.k8s.io_grpcroutes.yaml
var gatewayAPIGRPCRoutesCRD []byte

//go:embed gateway_api_crds/gateway.networking.k8s.io_httproutes.yaml
var gatewayAPIHTTPRoutesCRD []byte

//go:embed gateway_api_crds/gateway.networking.k8s.io_listenersets.yaml
var gatewayAPIListenerSetsCRD []byte

//go:embed gateway_api_crds/gateway.networking.k8s.io_referencegrants.yaml
var gatewayAPIReferenceGrantsCRD []byte

var gatewayAPICRDManifests = [][]byte{
	gatewayAPIBackendTLSPoliciesCRD,
	gatewayAPIGatewayClassesCRD,
	gatewayAPIGatewaysCRD,
	gatewayAPIGRPCRoutesCRD,
	gatewayAPIHTTPRoutesCRD,
	gatewayAPIListenerSetsCRD,
	gatewayAPIReferenceGrantsCRD,
}

type GatewayAPICRDs struct {
	shouldDelete bool
}

func NewGatewayAPICRDs(shouldDelete bool) GatewayAPICRDs {
	return GatewayAPICRDs{shouldDelete: shouldDelete}
}

func (GatewayAPICRDs) Name() string {
	return "GatewayAPICRDs"
}

func (g GatewayAPICRDs) reconcile(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	if g.shouldDelete {
		return g.deleteManagedCRDs(ctx, k8sClient)
	}
	return g.installOrWarnCRDs(ctx, k8sClient)
}

func (g GatewayAPICRDs) installOrWarnCRDs(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	var unmanagedCRDNames []string

	for _, manifest := range gatewayAPICRDManifests {
		var desired unstructured.Unstructured
		if err := yaml.Unmarshal(manifest, &desired); err != nil {
			return controllerutil.OperationResultNone, fmt.Errorf("failed to unmarshal Gateway API CRD manifest: %w", err)
		}

		existing := unstructured.Unstructured{}
		existing.SetGroupVersionKind(desired.GroupVersionKind())
		err := k8sClient.Get(ctx, client.ObjectKey{Name: desired.GetName()}, &existing)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return controllerutil.OperationResultNone, fmt.Errorf("failed to get CRD %s: %w", desired.GetName(), err)
			}
			// CRD does not exist — create it with module label
			applyManagementLabels(&desired)

			if createErr := k8sClient.Create(ctx, &desired); createErr != nil {
				return controllerutil.OperationResultNone, fmt.Errorf("failed to create CRD %s: %w", desired.GetName(), createErr)
			}
			if annotateErr := resources.AnnotateWithDisclaimer(ctx, &desired, k8sClient); annotateErr != nil {
				ctrl.Log.Error(annotateErr, "Failed to annotate Gateway API CRD with disclaimer", "crd", desired.GetName())
			}
			ctrl.Log.Info("Created Gateway API CRD", "name", desired.GetName())
			continue
		}

		// CRD already exists — check ownership
		if !isModuleManaged(existing) {
			ctrl.Log.Info("Gateway API CRD already exists and is not managed by Kyma Istio module, skipping", "name", existing.GetName())
			unmanagedCRDNames = append(unmanagedCRDNames, existing.GetName())
			continue
		}

		// Module-managed CRD — update it, preserving existing labels and annotations
		mergeIntoExisting(&desired, existing)

		if updateErr := k8sClient.Update(ctx, &desired); updateErr != nil {
			return controllerutil.OperationResultNone, fmt.Errorf("failed to update CRD %s: %w", desired.GetName(), updateErr)
		}
		if !resources.HasManagedByDisclaimer(existing) {
			if annotateErr := resources.AnnotateWithDisclaimer(ctx, &desired, k8sClient); annotateErr != nil {
				ctrl.Log.Error(annotateErr, "Failed to annotate Gateway API CRD with disclaimer", "crd", desired.GetName())
			}
		}
		ctrl.Log.Info("Updated Gateway API CRD", "name", desired.GetName())
	}

	if len(unmanagedCRDNames) > 0 {
		return controllerutil.OperationResultNone, &unmanagedCRDsWarning{names: unmanagedCRDNames}
	}
	return controllerutil.OperationResultUpdated, nil
}

func (g GatewayAPICRDs) deleteManagedCRDs(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	for _, manifest := range gatewayAPICRDManifests {
		var desired unstructured.Unstructured
		if err := yaml.Unmarshal(manifest, &desired); err != nil {
			return controllerutil.OperationResultNone, fmt.Errorf("failed to unmarshal Gateway API CRD manifest: %w", err)
		}

		existing := unstructured.Unstructured{}
		existing.SetGroupVersionKind(desired.GroupVersionKind())
		err := k8sClient.Get(ctx, client.ObjectKey{Name: desired.GetName()}, &existing)
		if err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return controllerutil.OperationResultNone, fmt.Errorf("failed to get CRD %s: %w", desired.GetName(), err)
		}

		if !isModuleManaged(existing) {
			ctrl.Log.Info("Skipping deletion of unmanaged Gateway API CRD", "name", existing.GetName())
			continue
		}

		if deleteErr := k8sClient.Delete(ctx, &existing); deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
			return controllerutil.OperationResultNone, fmt.Errorf("failed to delete CRD %s: %w", existing.GetName(), deleteErr)
		}
		ctrl.Log.Info("Deleted managed Gateway API CRD", "name", existing.GetName())
	}
	return controllerutil.OperationResultUpdated, nil
}

// isModuleManaged returns true if the resource carries the Kyma Istio module label.
func isModuleManaged(obj unstructured.Unstructured) bool {
	val, exists := obj.GetLabels()[labels.ModuleLabelKey]
	return exists && val == labels.ModuleLabelValue
}

// applyManagementLabels stamps the versioned label set and the module ownership label onto obj.
// Used on the create path where there is no existing object to preserve metadata from.
func applyManagementLabels(obj *unstructured.Unstructured) {
	resources.ApplyVersionedLabels(obj)
	lbls := obj.GetLabels()
	if lbls == nil {
		lbls = make(map[string]string)
	}
	lbls[labels.ModuleLabelKey] = labels.ModuleLabelValue
	obj.SetLabels(lbls)
}

// mergeIntoExisting prepares desired for an update by preserving existing labels and annotations
// and then enforcing the module-owned values on top. This prevents reconciliation from stripping
// labels/annotations set by other controllers (including the managed-by disclaimer).
func mergeIntoExisting(desired *unstructured.Unstructured, existing unstructured.Unstructured) {
	// Preserve existing labels, merge desired ones on top, then apply module-owned ones.
	lbls := make(map[string]string)
	for k, v := range existing.GetLabels() {
		lbls[k] = v
	}
	for k, v := range desired.GetLabels() {
		lbls[k] = v
	}
	desired.SetLabels(lbls)
	applyManagementLabels(desired)

	// Preserve existing annotations (e.g. the disclaimer), then merge desired ones on top.
	annotations := make(map[string]string)
	for k, v := range existing.GetAnnotations() {
		annotations[k] = v
	}
	for k, v := range desired.GetAnnotations() {
		annotations[k] = v
	}
	desired.SetAnnotations(annotations)

	desired.SetResourceVersion(existing.GetResourceVersion())
}

// unmanagedCRDsWarning is a soft error that signals pre-existing unmanaged CRDs.
type unmanagedCRDsWarning struct {
	names []string
}

func (e *unmanagedCRDsWarning) Error() string {
	return fmt.Sprintf("Gateway API CRDs already installed and not managed by Kyma Istio module: %v. "+
		"To allow Kyma Istio module to manage them, add the label %s=%s to each CRD",
		e.names, labels.ModuleLabelKey, labels.ModuleLabelValue)
}
