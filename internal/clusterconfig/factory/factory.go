package factory

type LB interface {
	Annotations() map[string]string
}

type CNI interface {
	CNIValues() map[string]interface{}
}

type Inputs struct {
	DualStackEnabled bool
	UsesGardenOS     bool
}

type Factory interface {
	LB() LB
	CNI() CNI
	NeedsProxyProtocol() bool
	DualStackEnabled() bool
}

type defaultFactory struct {
	inputs Inputs
}

// DefaultFactory is the fallback Factory for unknown cluster providers.
// It produces no LB or CNI customizations but reports the cluster's
// DualStackEnabled flag.
func DefaultFactory(in Inputs) Factory {
	return &defaultFactory{inputs: in}
}

func (f *defaultFactory) LB() LB                   { return nil }
func (f *defaultFactory) CNI() CNI                 { return nil }
func (f *defaultFactory) NeedsProxyProtocol() bool { return false }
func (f *defaultFactory) DualStackEnabled() bool   { return f.inputs.DualStackEnabled }
