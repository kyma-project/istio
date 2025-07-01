package resources

import (
	"context"
	"fmt"

	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/internal/describederrors"
)

// UserResourcesFinder is an interface that defines methods for detecting user-created resources in a Kubernetes cluster.
type UserResourcesFinder interface {
	// DetectUserCreatedEfOnIngress detects user-created EnvoyFilters that target istio-ingress-gateway.
	DetectUserCreatedEfOnIngress(ctx context.Context) describederrors.DescribedError
}

type UserResources struct {
	c client.Client
}

func NewUserResources(c client.Client) UserResources {
	return UserResources{
		c: c,
	}
}

func (urm UserResources) DetectUserCreatedEfOnIngress(ctx context.Context) describederrors.DescribedError {
	envoyFilterList := networkingv1alpha3.EnvoyFilterList{}

	err := urm.c.List(ctx, &envoyFilterList, client.InNamespace("istio-system"))
	if err != nil {
		return describederrors.NewDescribedError(err, "could not list EnvoyFilters")
	}
	for _, ef := range envoyFilterList.Items {
		if !isEfOwnedByRateLimit(ef) && !isEfOwnedByKymaModule(ef) && isTargetingIstioIngress(ef) {
			return describederrors.NewDescribedError(
				fmt.Errorf(
					"user-created EnvoyFilter %s/%s targeting Ingress Gateway found",
					ef.Namespace,
					ef.Name,
				),
				"misconfigured EnvoyFilter can potentially break Istio Ingress Gateway",
			).SetWarning()
		}
	}
	return nil
}

func isEfOwnedByRateLimit(ef *networkingv1alpha3.EnvoyFilter) bool {
	for _, owner := range ef.OwnerReferences {
		if owner.Kind == "RateLimit" {
			return true
		}
	}
	return false
}

func isEfOwnedByKymaModule(ef *networkingv1alpha3.EnvoyFilter) bool {
	_, ok := ef.Labels["kyma-project.io/module"]
	return ok
}

func isTargetingIstioIngress(ef *networkingv1alpha3.EnvoyFilter) bool {
	if ef.Spec.GetWorkloadSelector() == nil {
		return false
	}
	if ef.Namespace == "istio-system" &&
		(ef.Spec.GetWorkloadSelector().GetLabels()["istio"] == "ingressgateway" || ef.Spec.GetWorkloadSelector().GetLabels()["app"] == "istio-ingressgateway") {
		return true
	}
	return false
}
