package k3d

import (
	"github.com/kyma-project/istio/operator/internal/clusterconfig/strategy"
)

type K3D struct{}

func (s K3D) GetCNIValues() map[string]interface{} {
	return map[string]interface{}{
		"cniBinDir":  "/var/lib/rancher/k3s/data/cni",
		"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
	}
}

func NewStrategy() *strategy.Hyperscaler {
	return &strategy.Hyperscaler{
		CNI: K3D{},
	}
}
