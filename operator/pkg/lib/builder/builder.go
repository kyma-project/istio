package builder

import (
	"encoding/json"

	"istio.io/api/operator/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

type istioOperatorBuilder struct {
	istioOperator istioOperator.IstioOperator

	// Stored internally and returned on call to Get()
	mergeError error
}

func NewIstioOperatorBuilder(baseOperator ...istioOperator.IstioOperator) istioOperatorBuilder {
	newBuilder := istioOperatorBuilder{}
	if len(baseOperator) > 0 {
		newBuilder.istioOperator = baseOperator[0]
	} else {
		newBuilder.istioOperator = istioOperator.IstioOperator{
			Spec: &v1alpha1.IstioOperatorSpec{},
		}
	}

	return newBuilder
}

type Mergeable interface {
	// merge should merge any new configuration with the istioOperator parameter, overwriting if needed
	Merge(istioOperator.IstioOperator) (istioOperator.IstioOperator, error)
}

// MergeWith executes merge from supplied Mergeable with the builder stored IstioOperator as parameter
func (b *istioOperatorBuilder) MergeWith(toMerge ...Mergeable) *istioOperatorBuilder {
	for _, merge := range toMerge {
		out, err := merge.Merge(b.istioOperator)
		if err != nil {
			b.mergeError = err
			return b
		}
		b.istioOperator = out
	}
	return b
}

func (b *istioOperatorBuilder) GetString() (string, error) {
	if b.mergeError != nil {
		return "", b.mergeError
	}

	s, err := json.Marshal(b.istioOperator)
	if err != nil {
		return "", err
	}

	return string(s), nil
}

func (b *istioOperatorBuilder) GetJSONByteArray() ([]byte, error) {
	if b.mergeError != nil {
		return nil, b.mergeError
	}

	return json.Marshal(b.istioOperator)
}

func (b *istioOperatorBuilder) Get() (istioOperator.IstioOperator, error) {
	if b.mergeError != nil {
		return istioOperator.IstioOperator{}, b.mergeError
	}

	return b.istioOperator, nil
}
