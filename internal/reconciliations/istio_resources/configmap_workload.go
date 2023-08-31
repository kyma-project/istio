package istio_resources

import (
	"context"
	_ "embed"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed configmap_workload.yaml
var manifest_cm_workload []byte

type ConfigMapWorkload struct {
	k8sClient client.Client
}

func NewConfigMapWorkload(k8sClient client.Client) ConfigMapWorkload {
	return ConfigMapWorkload{k8sClient: k8sClient}
}

func (ConfigMapWorkload) apply(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	var resource unstructured.Unstructured
	err := yaml.Unmarshal(manifest_cm_workload, &resource)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	spec := resource.Object["spec"]
	result, err := controllerutil.CreateOrUpdate(ctx, k8sClient, &resource, func() error {
		resource.Object["spec"] = spec
		resource.SetOwnerReferences([]metav1.OwnerReference{owner})
		return nil
	})
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	var daFound bool
	if resource.GetAnnotations() != nil {
		_, daFound = resource.GetAnnotations()[istio.DisclaimerKey]
	}
	if !daFound {
		err := annotateWithDisclaimer(ctx, resource, k8sClient)
		if err != nil {
			return controllerutil.OperationResultNone, err
		}
	}

	return result, nil
}

func (ConfigMapWorkload) Name() string {
	return "ConfigMap/istio-workload-grafana-dashboard"
}
