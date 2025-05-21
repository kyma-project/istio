package resources

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// UserResourcesFinder is an interface that defines methods for detecting user-created resources in a Kubernetes cluster.
type UserResourcesFinder interface {
	// DetectUserCreatedEfOnIngress detects user-created EnvoyFilters that target istio-ingress-gateway.
	DetectUserCreatedEfOnIngress(ctx context.Context) described_errors.DescribedError
}

type UserResources struct {
	c client.Client
}

func NewUserResources(c client.Client) UserResources {
	return UserResources{
		c: c,
	}
}

func (urm UserResources) DetectUserCreatedEfOnIngress(ctx context.Context) described_errors.DescribedError {
	envoyFilterList := networkingv1alpha3.EnvoyFilterList{}

	err := urm.c.List(ctx, &envoyFilterList)
	if err != nil {
		return described_errors.NewDescribedError(err, "could not list EnvoyFilters")
	}
	for _, ef := range envoyFilterList.Items {
		if !isEfOwnedByRateLimit(ef) && isTargetingIstioIngress(ef) {
			return described_errors.NewDescribedError(
				fmt.Errorf("user-created EnvoyFilter %s/%s targeting Ingress Gateway found", ef.Namespace, ef.Name), "misconfigured EnvoyFilter can potentially break Istio Ingress Gateway").SetWarning()
		}
	}
	return nil
}

func isEfOwnedByRateLimit(ef *networkingv1alpha3.EnvoyFilter) bool {
	for _, owner := range ef.ObjectMeta.OwnerReferences {
		if owner.Kind == "RateLimit" {
			return true
		}
	}
	return false
}

func isTargetingIstioIngress(ef *networkingv1alpha3.EnvoyFilter) bool {
	if ef.Spec.WorkloadSelector == nil {
		return false
	}
	if ef.Spec.WorkloadSelector.Labels["istio"] == "ingressgateway" || ef.Spec.WorkloadSelector.Labels["app"] == "istio-ingressgateway" {
		return true
	}
	return false
}
