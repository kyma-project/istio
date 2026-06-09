package gke

import "github.com/kyma-project/istio/operator/internal/clusterconfig/strategy"

type CNI struct{}

func (CNI) GetCNIValues() (map[string]interface{}, bool) {
	return map[string]interface{}{
		"cniBinDir": "/home/kubernetes/bin",
		"resourceQuotas": map[string]bool{
			"enabled": true,
		},
	}, true
}

func NewStrategy() *strategy.Strategy {
	return &strategy.Strategy{
		CNI: &CNI{},
	}
}
