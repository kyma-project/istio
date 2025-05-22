package e2e

type Step interface {
	Name() string
	Args() map[string]string
	Execute() error
	AssertSuccess() error
	Output() interface{}
}
