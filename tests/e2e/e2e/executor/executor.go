package executor

import (
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	"github.com/stretchr/testify/assert"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
)

// Step defines a singular operation that can be executed as part of the test flow.
type Step interface {
	// Description returns a human-readable description of the step.
	// It might include details like the action being performed, the target resource, etc.
	// This is used for tracing of the test execution.
	Description() string
	// Execute runs the step, performing the necessary actions.
	// It takes a testing.T instance for logging and a client.Client for Kubernetes operations.
	Execute(*testing.T, client.Client) error
}

// Cleaner is an interface that defines a cleanup operation.
// A step should implement this interface if it requires cleanup after execution.
type Cleaner interface {
	// Cleanup performs the cleanup operation for the step.
	Cleanup(*testing.T, client.Client) error
}

// StepsStack is a structure that holds steps in a stack-like manner.
// It allows pushing new steps onto the stack and popping them off.
// This is useful for managing the order of cleanup.
type StepsStack struct {
	steps []Step
}

func NewStepsStack() *StepsStack {
	return &StepsStack{
		steps: make([]Step, 0),
	}
}

// empty returns true if the stack is empty.
func (q *StepsStack) empty() bool {
	return len(q.steps) == 0
}

// Push adds a new step to the top of the stack.
func (q *StepsStack) Push(step Step) {
	q.steps = append(q.steps, step)
}

// Pop removes and returns the step at the top of the stack.
func (q *StepsStack) Pop() Step {
	if len(q.steps) == 0 {
		return nil
	}
	step := q.steps[len(q.steps)-1]
	q.steps = q.steps[:len(q.steps)-1]
	return step
}

type Executor struct {
	Options

	t     *testing.T
	steps *StepsStack

	K8SClient client.Client
}

type Options struct {
	SkipCleanupAfterFailure bool
	OnlyPerformCleanup      bool
}

func NewExecutorWithOptionsFromEnv(t *testing.T) *Executor {
	options := Options{
		SkipCleanupAfterFailure: os.Getenv("SKIP_CLEANUP_AFTER_FAILURE") == "true",
		OnlyPerformCleanup:      os.Getenv("ONLY_PERFORM_CLEANUP") == "true",
	}

	return NewExecutor(t, options)
}

func NewExecutor(t *testing.T, options Options) *Executor {
	k8sClient, err := setup.ClientFromKubeconfig(t)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %s", err.Error())
	}

	return &Executor{
		t:         t,
		K8SClient: k8sClient,
		Options:   options,
		steps:     NewStepsStack(),
	}
}

// RunStep executes a step and adds it to the executor's cleanup stack.
func (e *Executor) RunStep(step Step) error {
	e.steps.Push(step)
	if e.OnlyPerformCleanup {
		logging.Debugf(e.t, "Skipping step execution in cleanup mode: %s", step.Description())
		return nil
	}

	logging.Tracef(e.t, "Starting execution of step: %s", step.Description())

	logging.Debugf(e.t, "Executing step: %s", step.Description())
	if err := step.Execute(e.t, e.K8SClient); err != nil {
		logging.Errorf(e.t, "Failed to execute step: %s err=%s", step.Description(), err.Error())
		return err
	}

	logging.Tracef(e.t, "Finished execution of step: %s", step.Description())
	return nil
}

// Cleanup performs the cleanup of all already executed steps in reverse order.
// If the executor is running in a CI environment (Options.SkipCleanupAfterFailure is true), cleanup is skipped.
// If the executor is in cleanup mode (Options.OnlyPerformCleanup is true),
// it will only perform cleanup without executing any steps.
func (e *Executor) Cleanup() {
	if e.SkipCleanupAfterFailure && !e.OnlyPerformCleanup && e.t.Failed() {
		logging.Infof(e.t, "Skipping cleanup due to SKIP_CLEANUP_AFTER_FAILURE being set")
		return
	}

	logging.Tracef(e.t, "Starting cleanup")
	defer logging.Tracef(e.t, "Finished cleanup")

	// Perform cleanup in reverse order
	for !e.steps.empty() {
		step := e.steps.Pop()
		if cleaner, ok := step.(Cleaner); ok {
			logging.Tracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
			err := cleaner.Cleanup(e.t, e.K8SClient)
			logging.Tracef(e.t, fmt.Sprintf("Finished cleaning up step: %s", step.Description()))
			if err != nil {
				assert.NoError(e.t, err)
			}
		} else {
			logging.Tracef(e.t, "Skipping cleanup for step: %s (not implementing Cleaner)", step.Description())
		}
	}
}
