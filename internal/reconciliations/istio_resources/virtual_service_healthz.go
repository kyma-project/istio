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

//go:embed virtual_service_healthz.yaml
var manifest_vs_healthz []byte

type VirtualServiceHealthz struct {
	k8sClient client.Client
}

func NewVirtualServiceHealthz(k8sClient client.Client) VirtualServiceHealthz {
	return VirtualServiceHealthz{k8sClient: k8sClient}
}

func (VirtualServiceHealthz) apply(ctx context.Context, k8sClient client.Client) (controllerutil.OperationResult, error) {
	var resource unstructured.Unstructured
	err := yaml.Unmarshal(manifest_vs_healthz, &resource)
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

func (VirtualServiceHealthz) Name() string {
	return "virtual service healthz"
}
