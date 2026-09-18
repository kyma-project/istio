package v2

import (
	"errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Continue indicates the Rule found no reason to act; evaluation should
	// proceed with the next Rule.
	Continue Decision = iota
	// Restart indicates the Object must be restarted.
	Restart
	// Stop indicates evaluation should halt without restarting.
	Stop
)

var (
	// ErrDecisionUnknown is returned when evaluated Rule returned unknown decision.
	ErrDecisionUnknown = errors.New("unknown decision")
)

// Decision represents the verdict a Rule produces when evaluating an Object.
type Decision int

// Object is the Kubernetes object a Rule evaluates. It aliases
// controller-runtime's client.Object.
type Object = client.Object

// Rule evaluates an Object and returns a Decision describing whether it should
// be restarted, skipped, or whether evaluation should stop.
type Rule interface {
	Evaluate(Object) Decision
}
