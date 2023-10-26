package istio_resources

import (
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/internal/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed configmap_control_plane.yaml
var manifest_cm_control_plane []byte

type ConfigMapControlPlane struct {
	k8sClient client.Client
}

func NewConfigMapControlPlane(k8sClient client.Client) ConfigMapControlPlane {
	return ConfigMapControlPlane{k8sClient: k8sClient}
}

func (ConfigMapControlPlane) apply(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	return resources.ApplyResource(ctx, k8sClient, manifest_cm_control_plane, &owner)
}

func (ConfigMapControlPlane) Name() string {
	return "ConfigMap/istio-control-plane-grafana-dashboard"
}
