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
	performanceDashboardName      = "istio-performance-grafana-dashboard"
	performanceDashboardNamespace = "kyma-system"
)

type ConfigMapPerformance struct {
	k8sClient client.Client
}

func NewConfigMapPerformance(k8sClient client.Client) ConfigMapPerformance {
	return ConfigMapPerformance{k8sClient: k8sClient}
}

func (ConfigMapPerformance) reconcile(ctx context.Context, k8sClient client.Client, owner metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	err := k8sClient.Delete(ctx, &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      performanceDashboardName,
		Namespace: performanceDashboardNamespace,
	}})

	if err != nil {
		if k8serrors.IsNotFound(err) {
			return controllerutil.OperationResultNone, nil
		}
		return "", err
	}
	return "deleted", nil
}

func (ConfigMapPerformance) Name() string {
	return "ConfigMap/istio-performance-grafana-dashboard"
}
