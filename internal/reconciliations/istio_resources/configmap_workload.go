package istio_resources

import (
	"context"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	workloadDashboardName      = "istio-workload-grafana-dashboard"
	workloadDashboardNamespace = "kyma-system"
)

type ConfigMapWorkload struct {
	k8sClient client.Client
}

func NewConfigMapWorkload(k8sClient client.Client) ConfigMapWorkload {
	return ConfigMapWorkload{k8sClient: k8sClient}
}

func (ConfigMapWorkload) reconcile(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	err := k8sClient.Delete(ctx, &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      workloadDashboardName,
		Namespace: workloadDashboardNamespace,
	}})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			return controllerutil.OperationResultNone, nil
		}
		return "", err
	}
	return "deleted", nil
}

func (ConfigMapWorkload) Name() string {
	return "ConfigMap/istio-workload-grafana-dashboard"
}
