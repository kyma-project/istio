package istio_resources

import (
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/internal/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed configmap_performance.yaml
var manifest_cm_performance []byte

type ConfigMapPerformance struct {
	k8sClient client.Client
}

func NewConfigMapPerformance(k8sClient client.Client) ConfigMapPerformance {
	return ConfigMapPerformance{k8sClient: k8sClient}
}

func (ConfigMapPerformance) apply(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	return resources.ApplyResource(ctx, k8sClient, manifest_cm_performance, &owner)
}

func (ConfigMapPerformance) Name() string {
	return "ConfigMap/istio-performance-grafana-dashboard"
}
