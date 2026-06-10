package strategy

type LB interface {
	GetLBAnnotations() (annotations map[string]string, needed bool)
	RequiresProxyProtocolEnvoyFilter() bool
}

type CNI interface {
	GetCNIValues() (values map[string]interface{}, needed bool)
}

type Hyperscaler struct {
	LB
	CNI
}
