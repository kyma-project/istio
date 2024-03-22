package istiooperator

import (
	_ "embed"
)

//go:embed istio-operator.yaml
var ProductionOperator []byte

//go:embed istio-operator-light.yaml
var EvaluationOperator []byte
