package istioresources

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"maps"

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

func parseCRD(path string) (unstructured.Unstructured, error) {
	data, err := gatewayAPICRDsFS.ReadFile(path)
	if err != nil {
		return unstructured.Unstructured{}, fmt.Errorf("failed to read embedded CRD %s: %w", path, err)
	}
	var desired unstructured.Unstructured
	if err := yaml.Unmarshal(data, &desired); err != nil {
		return unstructured.Unstructured{}, fmt.Errorf("failed to unmarshal Gateway API CRD manifest: %w", err)
	}
	return desired, nil
}

func (g GatewayAPICRDs) installOrWarnCRDs(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	managed, err := g.isGatewayManaged(ctx, k8sClient)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}
	if !managed {
		return controllerutil.OperationResultNone, &unmanagedCRDsWarning{}
	}

	if err := fs.WalkDir(gatewayAPICRDsFS, "gateway_api_crds", g.applyOne(ctx, k8sClient)); err != nil {
		return controllerutil.OperationResultNone, err
	}
	return controllerutil.OperationResultUpdated, nil
}

func (g GatewayAPICRDs) applyOne(ctx context.Context, k8sClient client.Client) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		desired, err := parseCRD(path)
		if err != nil {
			return err
		}

		existing := unstructured.Unstructured{}
		existing.SetGroupVersionKind(desired.GroupVersionKind())
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(&desired), &existing)
		if err != nil && !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to get CRD %s: %w", desired.GetName(), err)
		}

		if apierrors.IsNotFound(err) {
			applyManagementLabels(&desired)
			if createErr := k8sClient.Create(ctx, &desired); createErr != nil {
				return fmt.Errorf("failed to create CRD %s: %w", desired.GetName(), createErr)
			}
			logAnnotationError(ctx, &desired, k8sClient)
			ctrl.Log.Info("Created Gateway API CRD", "name", desired.GetName())
			return nil
		}

		mergeIntoExisting(&desired, existing)
		if updateErr := k8sClient.Update(ctx, &desired); updateErr != nil {
			return fmt.Errorf("failed to update CRD %s: %w", desired.GetName(), updateErr)
		}
		if !resources.HasManagedByDisclaimer(existing) {
			logAnnotationError(ctx, &desired, k8sClient)
		}
		ctrl.Log.Info("Updated Gateway API CRD", "name", desired.GetName())
		return nil
	}
}

func logAnnotationError(ctx context.Context, obj *unstructured.Unstructured, k8sClient client.Client) {
	if err := resources.AnnotateWithDisclaimer(ctx, obj, k8sClient); err != nil {
		ctrl.Log.Error(err, "Failed to annotate Gateway API CRD with disclaimer", "crd", obj.GetName())
	}
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
	if err := fs.WalkDir(gatewayAPICRDsFS, "gateway_api_crds", g.deleteOne(ctx, k8sClient)); err != nil {
		return controllerutil.OperationResultNone, err
	}
	return controllerutil.OperationResultUpdated, nil
}

func (g GatewayAPICRDs) deleteOne(ctx context.Context, k8sClient client.Client) fs.WalkDirFunc {
	return func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		desired, err := parseCRD(path)
		if err != nil {
			return err
		}

		existing := unstructured.Unstructured{}
		existing.SetGroupVersionKind(desired.GroupVersionKind())
		err = k8sClient.Get(ctx, client.ObjectKeyFromObject(&desired), &existing)
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
	}
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
	maps.Copy(lbls, existing.GetLabels())
	maps.Copy(lbls, desired.GetLabels())
	desired.SetLabels(lbls)
	applyManagementLabels(desired)

	annotations := make(map[string]string)
	maps.Copy(annotations, existing.GetAnnotations())
	maps.Copy(annotations, desired.GetAnnotations())
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
