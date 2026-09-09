package crds

import (
	"context"
	"errors"
	"fmt"

	"github.com/kyma-project/istio/operator/pkg/labels"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var gatewayAPICRDNames = []string{
	"backendtlspolicies.gateway.networking.k8s.io",
	"gatewayclasses.gateway.networking.k8s.io",
	"gateways.gateway.networking.k8s.io",
	"grpcroutes.gateway.networking.k8s.io",
	"httproutes.gateway.networking.k8s.io",
	"listenersets.gateway.networking.k8s.io",
	"referencegrants.gateway.networking.k8s.io",
}

var crdGVK = schema.GroupVersionKind{
	Group:   "apiextensions.k8s.io",
	Version: "v1",
	Kind:    "CustomResourceDefinition",
}

// AssertGatewayAPICRDsPresentWithModuleLabel checks that all Gateway API CRDs exist
// on the cluster and carry the kyma-project.io/module=istio label.
func AssertGatewayAPICRDsPresentWithModuleLabel(ctx context.Context, c client.Client) error {
	var errs []error
	for _, name := range gatewayAPICRDNames {
		crd := &unstructured.Unstructured{}
		crd.SetGroupVersionKind(crdGVK)
		if err := c.Get(ctx, types.NamespacedName{Name: name}, crd); err != nil {
			errs = append(errs, fmt.Errorf("CRD %s not found: %w", name, err))
			continue
		}
		val, exists := crd.GetLabels()[labels.ModuleLabelKey]
		if !exists || val != labels.ModuleLabelValue {
			errs = append(errs, fmt.Errorf("CRD %s missing module label %s=%s", name, labels.ModuleLabelKey, labels.ModuleLabelValue))
		}
	}
	return errors.Join(errs...)
}

// AssertGatewayAPICRDsNotPresent checks that none of the module-managed Gateway API CRDs
// exist on the cluster.
func AssertGatewayAPICRDsNotPresent(ctx context.Context, c client.Client) error {
	var found []string
	for _, name := range gatewayAPICRDNames {
		crd := &unstructured.Unstructured{}
		crd.SetGroupVersionKind(crdGVK)
		err := c.Get(ctx, types.NamespacedName{Name: name}, crd)
		if err == nil {
			found = append(found, name)
			continue
		}
		if !k8sErrors.IsNotFound(err) {
			return fmt.Errorf("unexpected error checking CRD %s: %w", name, err)
		}
	}
	if len(found) > 0 {
		return fmt.Errorf("expected Gateway API CRDs to be absent but found: %v", found)
	}
	return nil
}
