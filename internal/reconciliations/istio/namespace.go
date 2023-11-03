package istio

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/resources"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	istioNamespace   = "istio-system"
	wardenLabelKey   = "namespaces.warden.kyma-project.io/validate"
	wardenLabelValue = "enabled"
)

// addWardenValidationAndDisclaimer updates the Istio namespace
func addWardenValidationAndDisclaimer(ctx context.Context, kubeClient client.Client) error {
	ns := &v1.Namespace{}

	err := kubeClient.Get(ctx, types.NamespacedName{Name: istioNamespace}, ns)
	if err != nil {
		return err
	}
	patch := client.StrategicMergeFrom(ns.DeepCopy())
	ns.Annotations = addToMap(ns.Annotations, resources.DisclaimerKey, resources.DisclaimerValue)
	ns.Labels = addToMap(ns.Labels, wardenLabelKey, wardenLabelValue)

	err = kubeClient.Patch(ctx, ns, patch)
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
