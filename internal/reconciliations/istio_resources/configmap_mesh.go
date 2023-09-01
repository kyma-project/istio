package istio_resources

import (
	"context"
	_ "embed"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed configmap_mesh.yaml
var manifest_cm_mesh []byte

type ConfigMapMesh struct {
	k8sClient client.Client
}

func NewConfigMapMesh(k8sClient client.Client) ConfigMapMesh {
	return ConfigMapMesh{k8sClient: k8sClient}
}

func (ConfigMapMesh) apply(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	return applyResource(ctx, k8sClient, manifest_cm_mesh, &owner)
}

func (ConfigMapMesh) Name() string {
	return "ConfigMap/istio-mesh-grafana-dashboard"
}
