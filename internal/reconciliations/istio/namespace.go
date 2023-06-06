package istio

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	istioNamespace   = "istio-system"
	wardenLabelKey   = "namespaces.warden.kyma-project.io/validate"
	wardenLabelValue = "enabled"
	disclaimerKey    = "istios.operator.kyma-project.io/managed-by-disclaimer"
	disclaimerValue  = "DO NOT EDIT - This resource is managed by Kyma.\nAny modifications are discarded and the resource is reverted to the original state."
)

// addWardenValidationAndDisclaimer updates the Istio namespace
func addWardenValidationAndDisclaimer(ctx context.Context, kubeClient client.Client) error {
	var obj client.Object = &v1.Namespace{}

	err := kubeClient.Get(ctx, types.NamespacedName{Name: istioNamespace}, obj)
	if err != nil {
		return err
	}
	ns := obj.(*v1.Namespace)
	patch := client.StrategicMergeFrom(ns.DeepCopy())
	ns.Annotations = addToMap(ns.Annotations, disclaimerKey, disclaimerValue)
	ns.Labels = addToMap(ns.Labels, wardenLabelKey, wardenLabelValue)

	err = kubeClient.Patch(ctx, obj, patch)
	if err != nil {
		return err
	}

	return nil
}

func addToMap(labels map[string]string, key, val string) map[string]string {
	if len(labels) == 0 {
		labels = map[string]string{}
	}

	labels[key] = val
	return labels
}
