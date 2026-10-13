package istioresources

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

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

//go:embed gateway_api_crds/gateway.networking.k8s.io_*.yaml
var gatewayAPICRDsFS embed.FS

var (
	gatewayAPIGatewaysCRD, _       = gatewayAPICRDsFS.ReadFile("gateway_api_crds/gateway.networking.k8s.io_gateways.yaml")
	gatewayAPIGatewayClassesCRD, _ = gatewayAPICRDsFS.ReadFile("gateway_api_crds/gateway.networking.k8s.io_gatewayclasses.yaml")
)

// gatewayAPICRDName is the CRD checked to determine whether the module manages Gateway API CRDs.
const gatewayAPICRDName = "gateways.gateway.networking.k8s.io"

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

func walkCRDs(fn func(desired unstructured.Unstructured) error) error {
	return fs.WalkDir(gatewayAPICRDsFS, "gateway_api_crds", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := gatewayAPICRDsFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded CRD %s: %w", path, err)
		}
		var desired unstructured.Unstructured
		if err := yaml.Unmarshal(data, &desired); err != nil {
			return fmt.Errorf("failed to unmarshal Gateway API CRD manifest: %w", err)
		}
		return fn(desired)
	})
}

func (g GatewayAPICRDs) installOrWarnCRDs(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	managed, err := g.isGatewayManaged(ctx, k8sClient)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	if !managed {
		return controllerutil.OperationResultNone, &unmanagedCRDsWarning{}
	}

	if err := walkCRDs(func(desired unstructured.Unstructured) error {
		existing := unstructured.Unstructured{}
		existing.SetGroupVersionKind(desired.GroupVersionKind())
		err := k8sClient.Get(ctx, client.ObjectKey{Name: desired.GetName()}, &existing)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get CRD %s: %w", desired.GetName(), err)
			}
			applyManagementLabels(&desired)

			if createErr := k8sClient.Create(ctx, &desired); createErr != nil {
				return fmt.Errorf("failed to create CRD %s: %w", desired.GetName(), createErr)
			}
			if annotateErr := resources.AnnotateWithDisclaimer(ctx, &desired, k8sClient); annotateErr != nil {
				ctrl.Log.Error(annotateErr, "Failed to annotate Gateway API CRD with disclaimer", "crd", desired.GetName())
			}
			ctrl.Log.Info("Created Gateway API CRD", "name", desired.GetName())
			return nil
		}

		mergeIntoExisting(&desired, existing)

		if updateErr := k8sClient.Update(ctx, &desired); updateErr != nil {
			return fmt.Errorf("failed to update CRD %s: %w", desired.GetName(), updateErr)
		}
		if !resources.HasManagedByDisclaimer(existing) {
			if annotateErr := resources.AnnotateWithDisclaimer(ctx, &desired, k8sClient); annotateErr != nil {
				ctrl.Log.Error(annotateErr, "Failed to annotate Gateway API CRD with disclaimer", "crd", desired.GetName())
			}
		}
		ctrl.Log.Info("Updated Gateway API CRD", "name", desired.GetName())
		return nil
	}); err != nil {
		return controllerutil.OperationResultNone, err
	}

	return controllerutil.OperationResultUpdated, nil
}

// isGatewayManaged returns true if the gateway CRD exists and carries the managed-gateway-api label.
// Returns true (proceed) when the CRD does not exist yet — it will be created.
func (g GatewayAPICRDs) isGatewayManaged(ctx context.Context, k8sClient client.Client) (bool, error) {
	crd := unstructured.Unstructured{}
	crd.SetGroupVersionKind(crdGroupVersionKind)
	err := k8sClient.Get(ctx, client.ObjectKey{Name: gatewayAPICRDName}, &crd)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to get CRD %s: %w", gatewayAPICRDName, err)
	}
	return hasManagedGatewayAPILabel(crd), nil
}

func (g GatewayAPICRDs) deleteManagedCRDs(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	if err := walkCRDs(func(desired unstructured.Unstructured) error {
		existing := unstructured.Unstructured{}
		existing.SetGroupVersionKind(desired.GroupVersionKind())
		err := k8sClient.Get(ctx, client.ObjectKey{Name: desired.GetName()}, &existing)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return fmt.Errorf("failed to get CRD %s: %w", desired.GetName(), err)
		}

		if !hasModuleLabel(existing) {
			ctrl.Log.Info("Skipping deletion of unmanaged Gateway API CRD", "name", existing.GetName())
			return nil
		}

		if deleteErr := k8sClient.Delete(ctx, &existing); deleteErr != nil && !apierrors.IsNotFound(deleteErr) {
			return fmt.Errorf("failed to delete CRD %s: %w", existing.GetName(), deleteErr)
		}
		ctrl.Log.Info("Deleted managed Gateway API CRD", "name", existing.GetName())
		return nil
	}); err != nil {
		return controllerutil.OperationResultNone, err
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
	if obj.GetName() == gatewayAPICRDName {
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

// unmanagedCRDsWarning is a soft error used to signal the reconciler to return a Warning condition.
type unmanagedCRDsWarning struct{}

func (e *unmanagedCRDsWarning) Error() string {
	return fmt.Sprintf("Gateway API CRDs are already installed and not managed by Kyma Istio module. "+
		"To allow Kyma Istio module to manage them, add the label %s=%s to the %s CRD",
		labels.ManagedGatewayAPILabelKey, labels.ManagedGatewayAPILabelValue, gatewayAPICRDName)
}
