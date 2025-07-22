package resources

import (
	"context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

// variable is set to the correct version by the Dockerfile during build time.
var version = "dev"

// Apply creates or updates a resource in the given manifest in the cluster. The resource is annotated with a disclaimer.
// If the owner is provided, an OwnerReference is added to the resource. The function returns the operation result depending on the action taken.
// The function supports update operations on resources with spec and data fields.
func Apply(ctx context.Context, k8sClient client.Client, manifest []byte, owner *metav1.OwnerReference) (controllerutil.OperationResult, error) {
	resource, err := unmarshalManifest(manifest)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	ApplyVersionedLabels(&resource)

	resource, result, err := createOrUpdateResource(ctx, k8sClient, resource, owner)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	if !HasManagedByDisclaimer(resource) {
		annotateErr := AnnotateWithDisclaimer(ctx, &resource, k8sClient)
		if annotateErr != nil {
			return controllerutil.OperationResultNone, annotateErr
		}
	}

	return result, nil
}

func DeleteIfPresent(ctx context.Context, k8sClient client.Client, manifest []byte) (controllerutil.OperationResult, error) {
	resource, err := unmarshalManifest(manifest)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	err = k8sClient.Get(ctx, client.ObjectKeyFromObject(&resource), &resource)
	if err != nil {
		if !errors.IsNotFound(err) {
			return controllerutil.OperationResultNone, err
		}
		return controllerutil.OperationResultNone, nil
	}

	err = k8sClient.Delete(ctx, &resource)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	return controllerutil.OperationResultUpdated, nil
}

func ApplyVersionedLabels(resource *unstructured.Unstructured) {
	versionedLabels := resource.GetLabels()
	if versionedLabels == nil {
		versionedLabels = make(map[string]string)
	}
	versionedLabels["app.kubernetes.io/version"] = version
	resource.SetLabels(versionedLabels)
}

func unmarshalManifest(manifest []byte) (unstructured.Unstructured, error) {
	var resource unstructured.Unstructured

	err := yaml.Unmarshal(manifest, &resource)
	if err != nil {
		return resource, err
	}

	return resource, nil
}

func createOrUpdateResource(
	ctx context.Context,
	k8sClient client.Client,
	resource unstructured.Unstructured,
	owner *metav1.OwnerReference,
) (unstructured.Unstructured, controllerutil.OperationResult, error) {
	spec, specExist := resource.Object["spec"]
	data, dataExist := resource.Object["data"]
	labels := resource.GetLabels()
	result, err := controllerutil.CreateOrUpdate(ctx, k8sClient, &resource, func() error {
		l := resource.GetLabels()
		if l == nil {
			l = make(map[string]string)
		}
		for k, v := range labels {
			l[k] = v
		}
		resource.SetLabels(l)

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
