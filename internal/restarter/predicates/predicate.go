package predicates

import (
	"context"

	v1 "k8s.io/api/core/v1"
)

type SidecarProxyPredicate interface {
	Matches(v1.Pod) bool
	MustMatch() bool
}

type IngressGatewayPredicate interface {
	NewIngressGatewayEvaluator(context.Context) (IngressGatewayRestartEvaluator, error)
}

type IngressGatewayRestartEvaluator interface {
	// The RequiresIngressGatewayRestart method does not evaluate the restart per pod,
	// as there is only one Ingress Gateway deployment under Istio module control.
	RequiresIngressGatewayRestart() bool
}
