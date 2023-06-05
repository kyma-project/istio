package label

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
	disclaimerKey    = "istio.kyma-project.io/managed-by-istio-module-disclaimer"
	disclaimerValue  = "DO NOT EDIT - This resource is managed by Kyma.\nAny modifications are discarded and the resource is reverted to the original state."
)

func AddIstioNamespaceLabel(ctx context.Context, kubeClient client.Client) error {
	err := labelNamespace(ctx, kubeClient, istioNamespace, wardenLabelKey, wardenLabelValue)
	if err != nil {
		return err
	}
	err = labelNamespace(ctx, kubeClient, istioNamespace, disclaimerKey, disclaimerValue)
	if err != nil {
		return err
	}

	return nil
}

func labelNamespace(ctx context.Context, kubeClient client.Client, namespace, key, val string) error {
	obj := &v1.Namespace{}
	err := kubeClient.Get(ctx, types.NamespacedName{Name: namespace}, obj)
	if err != nil {
		return err
	}
	labels := addLabels(obj.Labels, key, val)
	obj.SetLabels(labels)

	patch := client.StrategicMergeFrom(obj.DeepCopy())
	err = kubeClient.Patch(ctx, obj, patch)
	if err != nil {
		return err
	}

	return nil
}

func addLabels(labels map[string]string, key, val string) map[string]string {
	if len(labels) == 0 {
		labels = map[string]string{}
	}

	labels[key] = val
	return labels
}
