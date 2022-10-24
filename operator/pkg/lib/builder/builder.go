package builder

import (
	"encoding/json"
	"log"

	"istio.io/api/operator/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

type istioOperatorBuilder struct {
	logger        *log.Logger
	istioOperator istioOperator.IstioOperator
}

func NewIstioOperatorBuilder(logger *log.Logger, baseOperator ...istioOperator.IstioOperator) istioOperatorBuilder {
	newBuilder := istioOperatorBuilder{}
	newBuilder.logger = logger
	if len(baseOperator) < 1 {
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
func (b *istioOperatorBuilder) MergeWith(toMerge ...Mergeable) (*istioOperatorBuilder, error) {
	for _, merge := range toMerge {
		out, err := merge.Merge(b.istioOperator)
		if err != nil {
			return nil, err
		}
		b.istioOperator = out
	}
	return b, nil
}

func (b *istioOperatorBuilder) String() string {
	s, err := json.Marshal(b.istioOperator)
	if err != nil {
		b.logger.Fatal(err)
	}

	return string(s)
}

func (b *istioOperatorBuilder) GetJSONByteArray() ([]byte, error) {
	return json.Marshal(b.istioOperator)
}

func (b *istioOperatorBuilder) Get() istioOperator.IstioOperator {
	return b.istioOperator
}
