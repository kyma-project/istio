package openstack

import "github.com/kyma-project/istio/operator/internal/clusterconfig/strategy"

const (
	proxyProtocolAnnotation = "loadbalancer.openstack.org/proxy-protocol"
	proxyProtocolVersion    = "v1"
)

type Strategy struct {
	isGardener bool
}

func (s Strategy) GetLBAnnotations() map[string]string {
	if s.isGardener {
		return map[string]string{
			proxyProtocolAnnotation: proxyProtocolVersion,
		}
	}
	return nil
}

func (s Strategy) RequiresProxyProtocolEnvoyFilter() bool {
	return true
}

func NewStrategy(isGardener bool) *strategy.Hyperscaler {
	return &strategy.Hyperscaler{
		LB: Strategy{
			isGardener: isGardener,
		},
	}
}
