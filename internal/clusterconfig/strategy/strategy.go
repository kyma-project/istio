package strategy

type LB interface {
	GetLBAnnotations() map[string]string
	RequiresProxyProtocolEnvoyFilter() bool
	IsDualStackEnabled() bool
}

type CNI interface {
	GetCNIValues() map[string]interface{}
}

type Hyperscaler struct {
	LB
	CNI
}
