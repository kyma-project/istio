package v2

import (
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Continue indicates the Evaluator found no reason to act; evaluation should
	// proceed with the next Evaluator.
	Continue Decision = iota
	// Restart indicates the Object must be restarted.
	Restart
	// Stop indicates evaluation should halt without restarting.
	Stop
)

var (
	// ErrDecisionUnknown is returned when evaluated Evaluator returned unknown decision.
	ErrDecisionUnknown = errors.New("unknown decision")
)

// Decision represents the verdict an Evaluator produces when evaluating an Object.
type Decision int

// Object is the Kubernetes object an Evaluator evaluates. It aliases
// controller-runtime's client.Object.
type Object = client.Object

// Evaluator evaluates an Object and returns a Decision describing whether it should
// be restarted, skipped, or whether evaluation should stop.
type Evaluator interface {
	Evaluate(Object) Decision
}
