package istio

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/kyma-project/istio/operator/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	gatewayAPIVersion        = "v1.4.1"
	ModuleLabelKey    string = "kyma-project.io/module"
	ModuleLabelValue  string = "istio"
)

//go:embed gateway-api-crds.yaml
var gatewayAPICRDsYAML string

type GatewayAPICRDInstaller struct {
	client client.Client
}

func NewGatewayAPICRDInstaller(c client.Client) *GatewayAPICRDInstaller {
	return &GatewayAPICRDInstaller{client: c}
}

// Install installs or updates Gateway API CRDs
func (g *GatewayAPICRDInstaller) Install(ctx context.Context) error {
	ctrl.Log.Info("Starting Gateway API CRDs installation", "version", gatewayAPIVersion)

	// Split the YAML file by document separator
	documents := strings.Split(gatewayAPICRDsYAML, "---")

	var createdCount, updatedCount, unchangedCount int

	for idx, doc := range documents {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}

		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := yaml.Unmarshal([]byte(doc), crd); err != nil {
			ctrl.Log.Error(err, "Failed to unmarshal Gateway API CRD", "documentIndex", idx)
			return fmt.Errorf("failed to unmarshal Gateway API CRD at document %d: %w", idx, err)
		}

		// Skip if not a valid CRD or not a Gateway API CRD
		if crd.Name == "" || !strings.Contains(crd.Name, "gateway.networking.k8s.io") {
			ctrl.Log.V(1).Info("Skipping non-Gateway API CRD document", "name", crd.Name, "documentIndex", idx)
			continue
		}

		// Check if CRD already exists
		existingCRD := &apiextensionsv1.CustomResourceDefinition{}
		err := g.client.Get(ctx, client.ObjectKey{Name: crd.Name}, existingCRD)

		if err != nil {
			if apierrors.IsNotFound(err) {
				// Create new CRD
				ctrl.Log.Info("Creating Gateway API CRD", "name", crd.Name, "version", gatewayAPIVersion)
				l := labels.SetModuleLabels(crd.GetLabels())
				crd.SetLabels(l)
				if err := g.client.Create(ctx, crd); err != nil {
					ctrl.Log.Error(err, "Failed to create Gateway API CRD", "name", crd.Name)
					return fmt.Errorf("failed to create Gateway API CRD %s: %w", crd.Name, err)
				}
				createdCount++
				ctrl.Log.Info("Successfully created Gateway API CRD", "name", crd.Name)
			} else {
				ctrl.Log.Error(err, "Failed to check Gateway API CRD existence", "name", crd.Name)
				return fmt.Errorf("failed to get Gateway API CRD %s: %w", crd.Name, err)
			}
		} else {
			existingLabels := existingCRD.GetLabels()
			moduleLabelVal, hasModuleLabel := existingLabels[labels.ModuleLabelKey]
			if !hasModuleLabel || moduleLabelVal != labels.ModuleLabelValue {
				ctrl.Log.Info("Existing Gateway API CRD is not managed by Kyma, skipping update", "name", crd.Name)
				unchangedCount++
				continue
			}

			existingVersion := "unknown"
			if val, ok := existingCRD.Annotations["gateway.networking.k8s.io/bundle-version"]; ok {
				existingVersion = val
			}

			targetVersion := gatewayAPIVersion
			if val, ok := crd.Annotations["gateway.networking.k8s.io/bundle-version"]; ok {
				targetVersion = val
			}

			// Only update if versions differ or if we need to ensure annotations are present
			if existingVersion != "unknown" && existingVersion == targetVersion {
				ctrl.Log.V(1).Info("Gateway API CRD is already up to date", "name", crd.Name, "version", existingVersion)
				unchangedCount++
			} else {
				// Log appropriately based on situation
				if existingVersion == "unknown" {
					ctrl.Log.Info("Ensuring Gateway API CRD is up to date", "name", crd.Name, "targetVersion", targetVersion)
				} else {
					ctrl.Log.Info("Updating Gateway API CRD", "name", crd.Name, "currentVersion", existingVersion, "targetVersion", targetVersion)
				}

				crd.ResourceVersion = existingCRD.ResourceVersion
				if err := g.client.Update(ctx, crd); err != nil {
					ctrl.Log.Error(err, "Failed to update Gateway API CRD", "name", crd.Name)
					return fmt.Errorf("failed to update Gateway API CRD %s: %w", crd.Name, err)
				}
				updatedCount++
				ctrl.Log.V(1).Info("Gateway API CRD update completed", "name", crd.Name, "version", targetVersion)
			}
		}
	}

	ctrl.Log.Info("Gateway API CRDs installation completed successfully",
		"version", gatewayAPIVersion,
		"created", createdCount,
		"updated", updatedCount,
		"unchanged", unchangedCount,
	)

	return nil
}

// Uninstall removes Gateway API CRDs (optional - can be used during cleanup)
func (g *GatewayAPICRDInstaller) Uninstall(ctx context.Context) error {
	ctrl.Log.Info("Starting Gateway API CRDs removal", "version", gatewayAPIVersion)

	documents := strings.Split(gatewayAPICRDsYAML, "---")

	var deletedCount, notFoundCount, failedCount int

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

		// Skip if not a Gateway API CRD
		if !strings.Contains(crd.Name, "gateway.networking.k8s.io") {
			continue
		}

		ctrl.Log.Info("Deleting Gateway API CRD", "name", crd.Name)
		if err := g.client.Delete(ctx, crd); err != nil {
			if apierrors.IsNotFound(err) {
				ctrl.Log.V(1).Info("Gateway API CRD already removed", "name", crd.Name)
				notFoundCount++
			} else {
				ctrl.Log.Error(err, "Failed to delete Gateway API CRD", "name", crd.Name)
				failedCount++
				// Continue with other CRDs even if one fails
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
		"failed", failedCount)
	return nil
}

// IsInstalled checks if Gateway API CRDs are installed
func (g *GatewayAPICRDInstaller) IsInstalled(ctx context.Context) (bool, error) {
	// Check for a key Gateway API CRD to determine if they are installed
	crdName := "gateways.gateway.networking.k8s.io"
	ctrl.Log.V(1).Info("Checking Gateway API CRDs installation status", "probeCRD", crdName)

	crd := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
	}

	err := g.client.Get(ctx, client.ObjectKey{Name: crd.Name}, crd)
	if err != nil {
		if apierrors.IsNotFound(err) {
			ctrl.Log.Info("Gateway API CRDs not found", "probeCRD", crdName)
			return false, nil
		}
		ctrl.Log.Error(err, "Failed to check Gateway API CRDs installation", "probeCRD", crdName)
		return false, err
	}

	installedVersion := "unknown"
	if val, ok := crd.Annotations["gateway.networking.k8s.io/bundle-version"]; ok {
		installedVersion = val
	}
	ctrl.Log.Info("Gateway API CRDs are installed", "probeCRD", crdName, "version", installedVersion)
	return true, nil
}
