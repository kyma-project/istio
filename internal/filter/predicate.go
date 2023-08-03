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
	RequiresIngressGatewayRestart(v1.Pod) bool
}
