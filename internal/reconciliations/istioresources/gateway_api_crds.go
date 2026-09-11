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
	"k8s.io/apimachinery/pkg/runtime/schema"
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

//go:embed gateway_api_crds/gateway.networking.k8s.io_tcproutes.yaml
var gatewayAPITCPRoutesCRD []byte

//go:embed gateway_api_crds/gateway.networking.k8s.io_tlsroutes.yaml
var gatewayAPITLSRoutesCRD []byte

//go:embed gateway_api_crds/gateway.networking.k8s.io_udproutes.yaml
var gatewayAPIUDPRoutesCRD []byte

var gatewayAPICRDManifests = [][]byte{
	gatewayAPIBackendTLSPoliciesCRD,
	gatewayAPIGatewayClassesCRD,
	gatewayAPIGatewaysCRD,
	gatewayAPIGRPCRoutesCRD,
	gatewayAPIHTTPRoutesCRD,
	gatewayAPIListenerSetsCRD,
	gatewayAPIReferenceGrantsCRD,
	gatewayAPITCPRoutesCRD,
	gatewayAPITLSRoutesCRD,
	gatewayAPIUDPRoutesCRD,
}

// gatewayAPIPrimaryCRDName is the single CRD checked to determine whether the module
// should take over management (or warn about unmanaged CRDs).
const gatewayAPIPrimaryCRDName = "gateways.gateway.networking.k8s.io"

var crdGroupVersionKind = schema.GroupVersionKind{
	Group:   "apiextensions.k8s.io",
	Version: "v1",
	Kind:    "CustomResourceDefinition",
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
	primaryUnmanaged, err := g.isPrimaryUnmanaged(ctx, k8sClient)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	if primaryUnmanaged {
		return controllerutil.OperationResultNone, &unmanagedCRDsWarning{name: gatewayAPIPrimaryCRDName}
	}

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

	return controllerutil.OperationResultUpdated, nil
}

// isPrimaryUnmanaged returns true if the primary CRD exists but lacks the managed-gateway-api label.
func (g GatewayAPICRDs) isPrimaryUnmanaged(ctx context.Context, k8sClient client.Client) (bool, error) {
	crd := unstructured.Unstructured{}
	crd.SetGroupVersionKind(crdGroupVersionKind)
	err := k8sClient.Get(ctx, client.ObjectKey{Name: gatewayAPIPrimaryCRDName}, &crd)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get CRD %s: %w", gatewayAPIPrimaryCRDName, err)
	}
	return !hasManagedGatewayAPILabel(crd), nil
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

		if !hasModuleLabel(existing) {
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

func hasManagedGatewayAPILabel(obj unstructured.Unstructured) bool {
	val, exists := obj.GetLabels()[labels.ManagedGatewayAPILabelKey]
	return exists && val == labels.ManagedGatewayAPILabelValue
}

func hasModuleLabel(obj unstructured.Unstructured) bool {
	val, exists := obj.GetLabels()[labels.ModuleLabelKey]
	return exists && val == labels.ModuleLabelValue
}

func applyManagementLabels(obj *unstructured.Unstructured) {
	resources.ApplyVersionedLabels(obj)
	lbls := obj.GetLabels()
	if lbls == nil {
		lbls = make(map[string]string)
	}
	lbls[labels.ModuleLabelKey] = labels.ModuleLabelValue
	if obj.GetName() == gatewayAPIPrimaryCRDName {
		lbls[labels.ManagedGatewayAPILabelKey] = labels.ManagedGatewayAPILabelValue
	}
	obj.SetLabels(lbls)
}

// mergeIntoExisting preserves existing labels/annotations and enforces module-owned values on top.
func mergeIntoExisting(desired *unstructured.Unstructured, existing unstructured.Unstructured) {
	lbls := make(map[string]string)
	for k, v := range existing.GetLabels() {
		lbls[k] = v
	}
	for k, v := range desired.GetLabels() {
		lbls[k] = v
	}
	desired.SetLabels(lbls)
	applyManagementLabels(desired)

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

// unmanagedCRDsWarning is a soft error signalling the primary Gateway API CRD is not module-managed.
type unmanagedCRDsWarning struct {
	name string
}

func (e *unmanagedCRDsWarning) Error() string {
	return fmt.Sprintf("Gateway API CRD %s is already installed and not managed by Kyma Istio module. "+
		"To allow Kyma Istio module to manage it, add the label %s=%s to the CRD",
		e.name, labels.ManagedGatewayAPILabelKey, labels.ManagedGatewayAPILabelValue)
}
