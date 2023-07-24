package filter

import v1 "k8s.io/api/core/v1"

type SidecarProxyPredicate interface {
	RequiresProxyRestart(v1.Pod) (bool, error)
}

type IngressGatewayPredicate interface {
	RequiresIngressGatewayRestart(v1.Pod) (bool, error)
}
