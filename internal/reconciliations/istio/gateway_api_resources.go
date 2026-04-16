package istio

import (
	"context"
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/pkg/labels"
)

// gatewayAPIKind pairs a GroupVersionKind with the name of its CRD resource.
type gatewayAPIKind struct {
	schema.GroupVersionKind
	// CRDName is the cluster-scoped CRD resource name (e.g. "gateways.gateway.networking.k8s.io").
	CRDName string
}

// gatewayAPIKinds lists all CR kinds installed via the Gateway API CRD bundle
// together with the name of the CRD that defines them.
var gatewayAPIKinds = []gatewayAPIKind{
	{GroupVersionKind: schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway"}, CRDName: "gateways.gateway.networking.k8s.io"},
	{GroupVersionKind: schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GatewayClass"}, CRDName: "gatewayclasses.gateway.networking.k8s.io"},
	{GroupVersionKind: schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"}, CRDName: "httproutes.gateway.networking.k8s.io"},
	{GroupVersionKind: schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GRPCRoute"}, CRDName: "grpcroutes.gateway.networking.k8s.io"},
	{GroupVersionKind: schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1beta1", Kind: "ReferenceGrant"}, CRDName: "referencegrants.gateway.networking.k8s.io"},
	{GroupVersionKind: schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1alpha3", Kind: "BackendTLSPolicy"}, CRDName: "backendtlspolicies.gateway.networking.k8s.io"},
}

// FindBlockingGatewayAPIResources returns all Gateway API CR instances whose parent CRD
// is managed by the Istio module (carries the kyma-project.io/module=istio label).
// CRs belonging to unmanaged CRDs are ignored – they cannot block module-owned CRD deletion.
// GVKs whose CRDs are not installed on the cluster are silently skipped.
func FindBlockingGatewayAPIResources(ctx context.Context, k8sClient client.Client) ([]string, error) {
	var blocking []string

	for _, entry := range gatewayAPIKinds {
		// Only CRs whose CRD is module-managed can block deletion.
		existingCRD := &apiextensionsv1.CustomResourceDefinition{}
		if err := k8sClient.Get(ctx, client.ObjectKey{Name: entry.CRDName}, existingCRD); err != nil {
			if errors.IsNotFound(err) {
				// CRD not on cluster at all – nothing to block.
				continue
			}
			return nil, fmt.Errorf("failed to get CRD %s: %w", entry.CRDName, err)
		}

		if existingCRD.GetLabels()[labels.ModuleLabelKey] != labels.ModuleLabelValue {
			// CRD exists but is not owned by the module – its CRs are not module concern.
			continue
		}

		list := &unstructured.UnstructuredList{}
		list.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   entry.Group,
			Version: entry.Version,
			Kind:    entry.Kind + "List",
		})

		if err := k8sClient.List(ctx, list); err != nil {
			if errors.IsNotFound(err) || meta.IsNoMatchError(err) {
				continue
			}
			return nil, fmt.Errorf("failed to list %s resources: %w", entry.Kind, err)
		}

		for _, item := range list.Items {
			blocking = append(blocking,
				fmt.Sprintf("%s %s/%s", entry.Kind, item.GetNamespace(), item.GetName()),
			)
		}
	}

	return blocking, nil
}
