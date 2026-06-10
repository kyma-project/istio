package gke

import "github.com/kyma-project/istio/operator/internal/clusterconfig/strategy"

type CNI struct{}

func (CNI) GetCNIValues() map[string]interface{} {
	return map[string]interface{}{
		"cniBinDir": "/home/kubernetes/bin",
		"resourceQuotas": map[string]bool{
			"enabled": true,
		},
	}
}

func NewStrategy() *strategy.Hyperscaler {
	return &strategy.Hyperscaler{
		CNI: &CNI{},
	}
}
