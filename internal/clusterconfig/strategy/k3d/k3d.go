package k3d

import (
	"github.com/kyma-project/istio/operator/internal/clusterconfig/strategy"
)

type K3D struct{}

func (s K3D) GetCNIValues() (map[string]interface{}, bool) {
	return map[string]interface{}{
		"cniBinDir":  "/var/lib/rancher/k3s/data/cni",
		"cniConfDir": "/var/lib/rancher/k3s/agent/etc/cni/net.d",
	}, true
}

func NewStrategy() *strategy.Strategy {
	return &strategy.Strategy{
		CNI: K3D{},
	}
}
