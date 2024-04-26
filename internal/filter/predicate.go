package filter

import (
	"context"
	v1 "k8s.io/api/core/v1"
)

type SidecarProxyPredicate interface {
	NewProxyRestartEvaluator(context.Context) (ProxyRestartEvaluator, error)
}

type IngressGatewayPredicate interface {
	NewIngressGatewayEvaluator(context.Context) (IngressGatewayRestartEvaluator, error)
}

type ProxyRestartEvaluator interface {
	RequiresProxyRestart(v1.Pod) bool
}

type IngressGatewayRestartEvaluator interface {
	// The RequiresIngressGatewayRestart method does not evaluate the restart per pod,
	//as there is only one Ingress Gateway deployment under Istio module control.
	RequiresIngressGatewayRestart() bool
}
