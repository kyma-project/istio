package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/kyma-project/istio/operator/pkg/labels"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var gatewayAPIResourceGVKs = []schema.GroupVersionKind{
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GatewayClass"},
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway"},
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"},
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GRPCRoute"},
	{Group: "gateway.networking.k8s.io", Version: "v1beta1", Kind: "ReferenceGrant"},
	{Group: "gateway.networking.k8s.io", Version: "v1alpha3", Kind: "BackendTLSPolicy"},
	{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "ListenerSet"},
}

type GatewayAPIResourcesFinder struct {
	ctx    context.Context
	client client.Client
}

func NewGatewayAPIResourcesFinder(ctx context.Context, k8sClient client.Client) *GatewayAPIResourcesFinder {
	return &GatewayAPIResourcesFinder{ctx: ctx, client: k8sClient}
}

// FindUserCreatedGatewayAPIResources returns all Gateway API resources on the cluster that are not managed by this module.
func (f *GatewayAPIResourcesFinder) FindUserCreatedGatewayAPIResources() ([]Resource, error) {
	var userResources []Resource

	for _, gvk := range gatewayAPIResourceGVKs {
		var list unstructured.UnstructuredList
		list.SetGroupVersionKind(gvk)
		if err := f.client.List(f.ctx, &list); err != nil {
			if errors.IsNotFound(err) || noMatchesForKind.MatchString(err.Error()) || couldNotFindReqResource.MatchString(err.Error()) {
				continue
			}
			return nil, fmt.Errorf("failed to list %s resources: %w", gvk.Kind, err)
		}

		for _, item := range list.Items {
			if labels.HasModuleLabels(item) {
				continue
			}
			if isIstioOwnedGatewayClass(item) {
				continue
			}
			userResources = append(userResources, Resource{
				GVK: gvk,
				ResourceMeta: ResourceMeta{
					Name:      item.GetName(),
					Namespace: item.GetNamespace(),
				},
			})
		}
	}

	return userResources, nil
}

// isIstioOwnedGatewayClass returns true for GatewayClass resources that Istio itself
// creates during installation (controllerName starts with "istio.io/").
// These are not user resources and must not block module deletion.
func isIstioOwnedGatewayClass(obj unstructured.Unstructured) bool {
	if obj.GetKind() != "GatewayClass" {
		return false
	}
	controllerName, _, _ := unstructured.NestedString(obj.Object, "spec", "controllerName")
	return strings.HasPrefix(controllerName, "istio.io/")
}

// HasAnyModuleManagedGatewayAPICRD returns true if the primary Gateway API CRD carries the managed-gateway-api label.
func HasAnyModuleManagedGatewayAPICRD(ctx context.Context, k8sClient client.Client) (bool, error) {
	crdGVK := schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	}
	crd := &unstructured.Unstructured{}
	crd.SetGroupVersionKind(crdGVK)
	if err := k8sClient.Get(ctx, client.ObjectKey{Name: "gateways.gateway.networking.k8s.io"}, crd); err != nil {
		if errors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check Gateway API CRD: %w", err)
	}
	val, exists := crd.GetLabels()[labels.ManagedGatewayAPILabelKey]
	return exists && val == labels.ManagedGatewayAPILabelValue, nil
}
