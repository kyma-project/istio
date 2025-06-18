package executor

import (
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
)

// TODO:
// Split interface executor and cleanuper
type Step interface {
	Description() string
	Execute(*testing.T, client.Client) error
}

type Cleanuper interface {
	Cleanup(*testing.T, client.Client) error
}

type StepsStack struct {
	steps []Step
}

func NewStepsStack() *StepsStack {
	return &StepsStack{
		steps: make([]Step, 0),
	}
}

func (q *StepsStack) Push(step Step) {
	q.steps = append(q.steps, step)
}

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
	IsCi        bool
	OnlyCleanup bool
}

func NewExecutorWithOptionsFromEnv(t *testing.T) *Executor {
	options := Options{
		IsCi:        os.Getenv("CI") == "true",
		OnlyCleanup: os.Getenv("ONLY_CLEANUP") == "true",
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

type RunStepOptions struct {
	RetryPeriod time.Duration
	Timeout     time.Duration
}

// RunStep executes a step and adds it to the executor's cleanup stack.
func (e *Executor) RunStep(step Step) error {
	e.steps.Push(step)
	if e.OnlyCleanup {
		logging.Debugf(e.t, "Skipping step execution in cleanup mode: %s", step.Description())
		return nil
	}

	logging.Tracef(e.t, step.Description())

	logging.Debugf(e.t, "Executing step: %s", step.Description())
	if err := step.Execute(e.t, e.K8SClient); err != nil {
		logging.Errorf(e.t, "Failed to execute step: %s err=%s", step.Description(), err.Error())
		return err
	}

	logging.Untracef(e.t, step.Description())
	return nil
}

func (e *Executor) Cleanup() {
	if e.IsCi && !e.OnlyCleanup {
		logging.Infof(e.t, "Skipping cleanup in CI environment")
		return
	}

	logging.Tracef(e.t, "Starting cleanup")
	defer logging.Untracef(e.t, "Finished cleanup")

	// Perform cleanup in reverse order
	for step := e.steps.Pop(); step != nil; step = e.steps.Pop() {
		if cleaner, ok := step.(Cleanuper); ok {
			logging.Tracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
			err := cleaner.Cleanup(e.t, e.K8SClient)
			logging.Untracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
			if err != nil {
				assert.NoError(e.t, err)
			}
		} else {
			logging.Tracef(e.t, "Skipping cleanup for step: %s (not implementing Cleanuper)", step.Description())
		}
	}
}
