package istio

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// gatewayAPIKinds lists all CRs installed via the Gateway API CRD bundle.
// These are the kinds from gateway-api-crds.yaml (gateway.networking.k8s.io).
var gatewayAPIKinds = []schema.GroupVersionKind{
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway"},
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GatewayClass"},
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"},
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GRPCRoute"},
	//TODO: reference grant is used? is needed?
	{Group: "gateway.networking.k8s.io", Version: "v1beta1", Kind: "ReferenceGrant"},
	{Group: "gateway.networking.k8s.io", Version: "v1alpha3", Kind: "BackendTLSPolicy"},
}

// FindUserCreatedGatewayAPIResources returns all Gateway API CRs present on the cluster.
// It skips GVKs whose CRDs are not installed (handles the case where enableGatewayAPI
// was true but CRDs were already partially cleaned up).
func FindUserCreatedGatewayAPIResources(ctx context.Context, k8sClient client.Client) ([]string, error) {
	var blocking []string

	for _, gvk := range gatewayAPIKinds {
		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   gvk.Group,
			Version: gvk.Version,
			Kind:    gvk.Kind + "List",
		})

		err := k8sClient.List(ctx, list)
		if err != nil {
			// CRD not installed on cluster - skip this GVK
			if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
				continue
			}
			return nil, fmt.Errorf("failed to list %s resources: %w", gvk.Kind, err)
		}

		for _, item := range list.Items {
			//TODO: is check for labels managed needed here?
			blocking = append(blocking,
				fmt.Sprintf("%s %s/%s", gvk.Kind, item.GetNamespace(), item.GetName()),
			)
		}
	}

	return blocking, nil
}
