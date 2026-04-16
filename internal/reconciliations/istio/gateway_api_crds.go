package istio

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	// Expected Gateway API CRDs version managed by Istio module. Single source of true for different CRDs.
	gatewayAPIVersion = "v1.4.1"
	gatewayAPIGroup   = "gateway.networking.k8s.io"
	bundleVersionKey  = "gateway.networking.k8s.io/bundle-version"
)

//go:embed gateway-api-crds.yaml
var gatewayAPICRDsYAML string

// GatewayAPICRDInstallResult holds the outcome of a single Install() call.
type GatewayAPICRDInstallResult struct {
	// CreatedCRDs lists CRD names that were freshly created and labelled.
	CreatedCRDs []string
	// UpdatedCRDs lists CRD names that were updated (version changed) and are managed.
	UpdatedCRDs []string
	// UnchangedCRDs lists CRD names that are already up to date and managed.
	UnchangedCRDs []string
	// UnmanagedCRDs lists CRD names that exist on the cluster but do NOT carry the
	// kyma-project.io/module=istio label. These were skipped – the user must add the
	// label manually for the module to manage them.
	UnmanagedCRDs []string
}

// HasUnmanagedCRDs returns true when at least one pre-existing CRD was found
// without the module-ownership label.
func (r GatewayAPICRDInstallResult) HasUnmanagedCRDs() bool {
	return len(r.UnmanagedCRDs) > 0
}

type GatewayAPICRDManager struct {
	client client.Client
}

func NewGatewayAPICRDManager(c client.Client) *GatewayAPICRDManager {
	return &GatewayAPICRDManager{client: c}
}

// Install installs or updates Gateway API CRDs.
// It returns a GatewayAPICRDInstallResult describing what happened to each CRD.
// Pre-existing CRDs without the module ownership label are recorded in UnmanagedCRDs
// and never modified – the caller is responsible for logging the situation to the user.
func (g *GatewayAPICRDManager) Install(ctx context.Context) (GatewayAPICRDInstallResult, error) {
	ctrl.Log.Info("Starting Gateway API CRDs installation", "version", gatewayAPIVersion)

	result := GatewayAPICRDInstallResult{}
	documents := strings.Split(gatewayAPICRDsYAML, "---")

	for idx, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal([]byte(doc), crd); err != nil {
			ctrl.Log.Error(err, "Failed to unmarshal Gateway API CRD", "documentIndex", idx)
			return result, fmt.Errorf("failed to unmarshal Gateway API CRD at document %d: %w", idx, err)
		}

		if crd.Name == "" || !strings.Contains(crd.Name, gatewayAPIGroup) {
			ctrl.Log.V(1).Info("Skipping non-Gateway API CRD document", "name", crd.Name, "documentIndex", idx)
			continue
		}

		existingCRD := &apiextensionsv1.CustomResourceDefinition{}
		err := g.client.Get(ctx, client.ObjectKey{Name: crd.Name}, existingCRD)

		targetVersion := crd.Annotations[bundleVersionKey]

		if apierrors.IsNotFound(err) {
			ctrl.Log.Info("Creating Gateway API CRD", "name", crd.Name, "version", targetVersion)
			crd.SetLabels(labels.SetModuleLabels(crd.GetLabels()))
			if createErr := g.client.Create(ctx, crd); createErr != nil {
				ctrl.Log.Error(createErr, "Failed to create Gateway API CRD", "name", crd.Name)
				return result, fmt.Errorf("failed to create Gateway API CRD %s: %w", crd.Name, createErr)
			}
			result.CreatedCRDs = append(result.CreatedCRDs, crd.Name)
			ctrl.Log.Info("Successfully created Gateway API CRD", "name", crd.Name)
			continue
		}

		if err != nil {
			ctrl.Log.Error(err, "Failed to check Gateway API CRD existence", "name", crd.Name)
			return result, fmt.Errorf("failed to get Gateway API CRD %s: %w", crd.Name, err)
		}

		// CRD already exists – check ownership before touching it.
		if existingCRD.GetLabels()[labels.ModuleLabelKey] != labels.ModuleLabelValue {
			ctrl.Log.Info("Gateway API CRD exists but is not managed by the Istio module, skipping",
				"name", crd.Name,
				"hint", fmt.Sprintf("add label %s=%s to allow the Istio module to manage this CRD", labels.ModuleLabelKey, labels.ModuleLabelValue),
			)
			result.UnmanagedCRDs = append(result.UnmanagedCRDs, crd.Name)
			continue
		}

		// CRD already exists and is managed by Istio module.
		// Check CRD version
		existingVersion := existingCRD.Annotations[bundleVersionKey]
		// TODO: is it needed? what is the case for that? how to manage that?
		if targetVersion != gatewayAPIVersion {
			ctrl.Log.Info("Gateway API CRD",
				"name", crd.Name, "targetVersion", targetVersion, "is different from expected version", gatewayAPIVersion)
		}

		if existingVersion != "" && existingVersion == targetVersion {
			ctrl.Log.V(1).Info("Gateway API CRD is already up to date", "name", crd.Name, "version", existingVersion)
			result.UnchangedCRDs = append(result.UnchangedCRDs, crd.Name)
			continue
		}

		// TODO: look into that
		if existingVersion == "" {
			ctrl.Log.Info("Ensuring Gateway API CRD is up to date (no bundle-version annotation found)",
				"name", crd.Name, "targetVersion", targetVersion)
		} else {
			ctrl.Log.Info("Updating Gateway API CRD",
				"name", crd.Name, "currentVersion", existingVersion, "targetVersion", targetVersion)
		}

		crd.SetLabels(labels.SetModuleLabels(crd.GetLabels()))
		// Preventing silent data races. If two actors try to update the same CRD simultaneously,
		// only the one whose ResourceVersion still matches the cluster wins.
		crd.ResourceVersion = existingCRD.ResourceVersion
		if updateErr := g.client.Update(ctx, crd); updateErr != nil {
			ctrl.Log.Error(updateErr, "Failed to update Gateway API CRD", "name", crd.Name)
			return result, fmt.Errorf("failed to update Gateway API CRD %s: %w", crd.Name, updateErr)
		}
		result.UpdatedCRDs = append(result.UpdatedCRDs, crd.Name)
		ctrl.Log.V(1).Info("Gateway API CRD update completed", "name", crd.Name, "version", targetVersion)
	}

	ctrl.Log.Info("Gateway API CRDs installation completed",
		"version", gatewayAPIVersion,
		"created", len(result.CreatedCRDs),
		"updated", len(result.UpdatedCRDs),
		"unchanged", len(result.UnchangedCRDs),
		"unmanaged_skipped", len(result.UnmanagedCRDs),
	)
	return result, nil
}

// Uninstall removes only the Gateway API CRDs that carry the module ownership label
// (kyma-project.io/module=istio). CRDs without that label are left untouched.
//
// When statusHandler and istioCR are provided (non-nil), the function checks for
// existing Gateway API custom resources before deleting each managed CRD. If any are
// found it sets ConditionReasonGatewayAPICRsDangling on the Istio CR, logs each
// blocking resource, and returns an error so the caller can halt the uninstallation.
func (g *GatewayAPICRDManager) Uninstall(
	ctx context.Context,
	statusHandler status.Status,
	istioCR *operatorv1alpha2.Istio,
) error {
	ctrl.Log.Info("Starting Gateway API CRDs removal (labelled CRDs only)", "version", gatewayAPIVersion)

	documents := strings.Split(gatewayAPICRDsYAML, "---")
	var deletedCount, notFoundCount, skippedCount int

	for _, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal([]byte(doc), crd); err != nil {
			ctrl.Log.V(1).Info("Skipping document that failed to unmarshal during uninstall")
			continue
		}

		if !strings.Contains(crd.Name, gatewayAPIGroup) {
			continue
		}

		// Fetch the live object to check the ownership label before deleting.
		existingCRD := &apiextensionsv1.CustomResourceDefinition{}
		if getErr := g.client.Get(ctx, client.ObjectKey{Name: crd.Name}, existingCRD); getErr != nil {
			if apierrors.IsNotFound(getErr) {
				ctrl.Log.V(1).Info("Gateway API CRD already absent, nothing to remove", "name", crd.Name)
				notFoundCount++
				continue
			}
			ctrl.Log.Error(getErr, "Failed to fetch Gateway API CRD during uninstall, skipping", "name", crd.Name)
			continue
		}

		if existingCRD.GetLabels()[labels.ModuleLabelKey] != labels.ModuleLabelValue {
			ctrl.Log.Info("Gateway API CRD is not managed by the Istio module, skipping deletion", "name", crd.Name)
			skippedCount++
			continue
		}

		// CRD is managed – check for blocking CRs before deleting.
		blocking, err := FindUserCreatedGatewayAPIResources(ctx, g.client)
		if err != nil {
			ctrl.Log.Error(err, "Failed to check for blocking Gateway API resources", "crd", crd.Name)
			return fmt.Errorf("could not check for Gateway API resources before deleting CRD %s: %w", crd.Name, err)
		}
		if len(blocking) > 0 {
			for _, r := range blocking {
				ctrl.Log.Info("Gateway API resource is blocking CRD deletion", "resource", r, "crd", crd.Name)
			}
			msg := fmt.Sprintf(
				"Gateway API CRD deletion blocked by %d existing Gateway API custom resources. Remove them first.",
				len(blocking),
			)
			// TODO: Check if tatusHandler (interface/value) — if status.Status is an interface backed by a pointer receiver, mutations inside Uninstall will persist. If it's a value type passed by copy, changes inside Uninstall won't be visible outside. You should verify this in status.Status definition.
			statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(
				operatorv1alpha2.ConditionReasonGatewayAPICRsDangling, msg,
			))
			return fmt.Errorf("cannot delete Gateway API CRD %s: %d Gateway API resources are still present on the cluster", crd.Name, len(blocking))
		}

		ctrl.Log.Info("Deleting managed Gateway API CRD", "name", crd.Name)
		if deleteErr := g.client.Delete(ctx, existingCRD); deleteErr != nil {
			if apierrors.IsNotFound(deleteErr) {
				ctrl.Log.V(1).Info("Gateway API CRD already removed", "name", crd.Name)
				notFoundCount++
			} else {
				ctrl.Log.Error(deleteErr, "Failed to delete Gateway API CRD", "name", crd.Name)
			}
		} else {
			ctrl.Log.Info("Successfully deleted Gateway API CRD", "name", crd.Name)
			deletedCount++
		}
	}

	ctrl.Log.Info("Gateway API CRDs removal completed",
		"version", gatewayAPIVersion,
		"deleted", deletedCount,
		"notFound", notFoundCount,
		"skipped_unmanaged", skippedCount,
	)
	return nil
}
