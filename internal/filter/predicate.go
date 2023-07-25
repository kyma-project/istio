package filter

import (
	"context"
	v1 "k8s.io/api/core/v1"
)

type SidecarProxyPredicate interface {
	RequiresProxyRestart(context.Context, v1.Pod) (bool, error)
}

type IngressGatewayPredicate interface {
	RequiresIngressGatewayRestart(context.Context, v1.Pod) (bool, error)
}
