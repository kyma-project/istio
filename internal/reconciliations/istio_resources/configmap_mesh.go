package istio_resources

import (
	"context"
	_ "embed"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed configmap_mesh.yaml
var manifest_cm_mesh []byte

type ConfigMapMesh struct {
	k8sClient client.Client
}

func NewConfigMapMesh(k8sClient client.Client) ConfigMapMesh {
	return ConfigMapMesh{k8sClient: k8sClient}
}

func (ConfigMapMesh) apply(ctx context.Context, k8sClient client.Client, _ map[string]string) (controllerutil.OperationResult, error) {
	var resource unstructured.Unstructured
	err := yaml.Unmarshal(manifest_cm_mesh, &resource)
	if err != nil {
		return controllerutil.OperationResultNone, err
	}

	spec := resource.Object["spec"]
	result, err := controllerutil.CreateOrUpdate(ctx, k8sClient, &resource, func() error {
		resource.Object["spec"] = spec
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

func (ConfigMapMesh) Name() string {
	return "config map mesh"
}
