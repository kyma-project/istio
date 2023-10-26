package resources

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// Apply creates or updates a resource in the given manifest in the cluster. The resource is annotated with a disclaimer.
// If the owner is provided, a OwnerReference is added to the resource. The function returns the operation result depending on the action taken.
// The function supports update operations on resources with spec and data fields.
func Apply(ctx context.Context, k8sClient client.Client, manifest []byte, owner *metav1.OwnerReference) (controllerutil.OperationResult, error) {
	resource, err := unmarshalManifest(manifest)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	resource, result, err := createOrUpdateResource(ctx, k8sClient, resource, owner)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if !hasManagedByDisclaimer(resource) {
		err := AnnotateWithDisclaimer(ctx, resource, k8sClient)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
	}

	return result, nil
}

func unmarshalManifest(manifest []byte) (unstructured.Unstructured, error) {
	var resource unstructured.Unstructured

	err := yaml.Unmarshal(manifest, &resource)
	if err != nil {
		return resource, err
	}

	return resource, nil
}

func createOrUpdateResource(ctx context.Context, k8sClient client.Client, resource unstructured.Unstructured, owner *metav1.OwnerReference) (unstructured.Unstructured, controllerutil.OperationResult, error) {
	spec, specExist := resource.Object["spec"]
	data, dataExist := resource.Object["data"]
	result, err := controllerutil.CreateOrUpdate(ctx, k8sClient, &resource, func() error {
		if dataExist {
			resource.Object["data"] = data
		}

		if specExist {
			resource.Object["spec"] = spec
		}

		if owner != nil {
			resource.SetOwnerReferences([]metav1.OwnerReference{*owner})
		}
		return nil
	})

	return resource, result, err
}
