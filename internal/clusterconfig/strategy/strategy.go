package strategy

type LB interface {
	GetLBAnnotations() map[string]string
	RequiresProxyProtocolEnvoyFilter() bool
}

type CNI interface {
	GetCNIValues() map[string]interface{}
}

type Hyperscaler struct {
	LB
	CNI
	// DualStackEnabled mirrors the cluster-wide kyma-provisioning-info dualStackIPEnabled flag.
	// It is independent of the LB strategy.
	DualStackEnabled bool
}
