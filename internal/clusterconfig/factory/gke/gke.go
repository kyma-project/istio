package gke

import (
	"github.com/kyma-project/istio/operator/internal/clusterconfig/factory"
)

type CNI struct{}

func (CNI) GetCNIValues() map[string]interface{} {
	return map[string]interface{}{
		"cniBinDir": "/home/kubernetes/bin",
		"resourceQuotas": map[string]bool{
			"enabled": true,
		},
	}
}

type Factory struct {
	inputs factory.Inputs
}

func NewFactory(in factory.Inputs) *Factory { return &Factory{inputs: in} }

func (f *Factory) LB() factory.LB           { return nil }
func (f *Factory) CNI() factory.CNI         { return CNI{} }
func (f *Factory) NeedsProxyProtocol() bool { return false }
func (f *Factory) DualStackEnabled() bool   { return f.inputs.DualStackEnabled }
