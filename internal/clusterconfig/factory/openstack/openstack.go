package openstack

import "github.com/kyma-project/istio/operator/internal/clusterconfig/factory"

const (
	proxyProtocolAnnotation = "loadbalancer.openstack.org/proxy-protocol"
	proxyProtocolVersion    = "v1"
)

type LB struct {
	isGardener bool
}

func (s LB) Annotations() map[string]string {
	if s.isGardener {
		return map[string]string{
			proxyProtocolAnnotation: proxyProtocolVersion,
		}
	}
	return nil
}

type Factory struct {
	inputs factory.Inputs
}

func NewFactory(in factory.Inputs) *Factory { return &Factory{inputs: in} }

func (f *Factory) LB() factory.LB {
	return LB{isGardener: f.inputs.UsesGardenOS}
}
func (f *Factory) CNI() factory.CNI         { return nil }
func (f *Factory) NeedsProxyProtocol() bool { return true }
func (f *Factory) DualStackEnabled() bool   { return f.inputs.DualStackEnabled }
