package builder_test

import (
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

// FakeMergeable implements Mergeable for the sake of testing
type FakeMergeable struct {
	// If ThrowError is not nil it will be returned from Merge
	ThrowError error
	// Merge will change IstioOperator.Spec.Namespace to NewNamespaceName
	NewNamespaceName string
}

func (f FakeMergeable) Merge(op istioOperator.IstioOperator) (istioOperator.IstioOperator, error) {
	op.Spec.Namespace = f.NewNamespaceName
	return op, f.ThrowError
}
