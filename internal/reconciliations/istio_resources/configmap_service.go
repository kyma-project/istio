package istio_resources

import (
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/internal/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed configmap_service.yaml
var manifest_cm_service []byte

type ConfigMapService struct {
	k8sClient client.Client
}

func NewConfigMapService(k8sClient client.Client) ConfigMapService {
	return ConfigMapService{k8sClient: k8sClient}
}

func (ConfigMapService) apply(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	return resources.ApplyResource(ctx, k8sClient, manifest_cm_service, &owner)
}

func (ConfigMapService) Name() string {
	return "ConfigMap/istio-service-grafana-dashboard"
}
