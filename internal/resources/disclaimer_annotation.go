package resources

import (
	"context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	DisclaimerKey   = "istios.operator.kyma-project.io/managed-by-disclaimer"
	DisclaimerValue = "DO NOT EDIT - This resource is managed by Kyma.\nAny modifications are discarded and the resource is reverted to the original state."
)

func AnnotateWithDisclaimer(ctx context.Context, resource unstructured.Unstructured, k8sClient client.Client) error {
	annotations := resource.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[DisclaimerKey] = DisclaimerValue
	resource.SetAnnotations(annotations)

	err := k8sClient.Update(ctx, &resource)
	return err
}

func hasManagedByDisclaimer(resource unstructured.Unstructured) bool {
	if resource.GetAnnotations() != nil {
		_, daFound := resource.GetAnnotations()[DisclaimerKey]
		return daFound
	}

	return false
}
