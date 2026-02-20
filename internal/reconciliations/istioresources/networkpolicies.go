package istioresources

import (
	"context"
	_ "embed"

	"github.com/kyma-project/istio/operator/internal/resources"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:embed networkpolicies/allow-cni.yaml
var allowCni []byte

//go:embed networkpolicies/allow-egress-to-customer.yaml
var allowEgressToCustomer []byte

//go:embed networkpolicies/allow-ingressgateway.yaml
var allowIngressGateway []byte

//go:embed networkpolicies/allow-istio-controller-manager.yaml
var allowIstioControllerManager []byte

//go:embed networkpolicies/allow-istiod.yaml
var allowIstiod []byte

//go:embed networkpolicies/allow-jwks.yaml
var allowJwks []byte

type NetworkPolicies struct {
	shouldDelete bool
}

func NewNetworkPolicies(shouldDelete bool) NetworkPolicies {
	return NetworkPolicies{
		shouldDelete: shouldDelete,
	}
}

func (NetworkPolicies) Name() string {
	return "NetworkPolicies"
}

func (np NetworkPolicies) reconcile(ctx context.Context, k8sClient client.Client, _ metav1.OwnerReference, _ map[string]string) (controllerutil.OperationResult, error) {
	if np.shouldDelete {
		result, err := resources.DeleteIfPresent(ctx, k8sClient, allowCni)
		if err != nil {
			return result, err
		}
		result, err = resources.DeleteIfPresent(ctx, k8sClient, allowEgressToCustomer)
		if err != nil {
			return result, err
		}
		result, err = resources.DeleteIfPresent(ctx, k8sClient, allowIngressGateway)
		if err != nil {
			return result, err
		}
		result, err = resources.DeleteIfPresent(ctx, k8sClient, allowIstioControllerManager)
		if err != nil {
			return result, err
		}
		result, err = resources.DeleteIfPresent(ctx, k8sClient, allowIstiod)
		if err != nil {
			return result, err
		}
		return resources.DeleteIfPresent(ctx, k8sClient, allowJwks)
	}
	result, err := resources.Apply(ctx, k8sClient, allowCni, nil)
	if err != nil {
		return result, err
	}
	result, err = resources.Apply(ctx, k8sClient, allowEgressToCustomer, nil)
	if err != nil {
		return result, err
	}
	result, err = resources.Apply(ctx, k8sClient, allowIngressGateway, nil)
	if err != nil {
		return result, err
	}
	result, err = resources.Apply(ctx, k8sClient, allowIstioControllerManager, nil)
	if err != nil {
		return result, err
	}
	result, err = resources.Apply(ctx, k8sClient, allowIstiod, nil)
	if err != nil {
		return result, err
	}
	return resources.Apply(ctx, k8sClient, allowJwks, nil)
}
