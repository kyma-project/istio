package istio_resources

import (
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/internal/resources"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	return resources.Apply(ctx, k8sClient, manifest_cm_workload, &owner)
}

func (ConfigMapWorkload) Name() string {
	return "ConfigMap/istio-workload-grafana-dashboard"
}
