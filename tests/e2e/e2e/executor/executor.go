package executor

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
)

type Step interface {
	Description() string
	Execute(*testing.T, client.Client) error
	Cleanup(*testing.T, client.Client) error
}

type StepsLIFOQueue struct {
	steps []Step
}

func NewStepLIFOQueue() *StepsLIFOQueue {
	return &StepsLIFOQueue{
		steps: make([]Step, 0),
	}
}

func (q *StepsLIFOQueue) Push(step Step) {
	q.steps = append(q.steps, step)
}

func (q *StepsLIFOQueue) Pop() Step {
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
	steps *StepsLIFOQueue

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
		steps:     NewStepLIFOQueue(),
	}
}

const (
	ErrorPrefix   = "[ERROR] "
	InfoPrefix    = "[INFO] "
	TracePrefix   = "[TRACE] "
	UntracePrefix = "[UNTRACE] "
	DebugPrefix   = "[DEBUG] "
)

func Errorf(t *testing.T, template string, args ...interface{}) {
	t.Logf(ErrorPrefix+template, args...)
}

func Infof(t *testing.T, template string, args ...interface{}) {
	t.Logf(InfoPrefix+template, args...)
}

func Debugf(t *testing.T, template string, args ...interface{}) {
	t.Logf(DebugPrefix+template, args...)
}

func Tracef(t *testing.T, template string, args ...interface{}) {
	t.Logf(TracePrefix+template, args...)
}

func Untracef(t *testing.T, template string, args ...interface{}) {
	t.Logf(UntracePrefix+template, args...)
}

func (e *Executor) RunStep(step Step) error {
	e.steps.Push(step)
	if e.OnlyCleanup {
		Debugf(e.t, "Skipping step execution in cleanup mode: %s", step.Description())
		return nil
	}

	Tracef(e.t, step.Description())

	if err := step.Execute(e.t, e.K8SClient); err != nil {
		Errorf(e.t, "Failed to execute step: %s err=%s", step.Description(), err.Error())
		return err
	}

	Untracef(e.t, step.Description())
	return nil
}

func (e *Executor) Cleanup() {
	if e.IsCi && !e.OnlyCleanup {
		Infof(e.t, "Skipping cleanup in CI environment")
		return
	}

	Tracef(e.t, "Starting cleanup")
	defer Untracef(e.t, "Finished cleanup")

	// Perform cleanup in reverse order
	for step := e.steps.Pop(); step != nil; step = e.steps.Pop() {
		Tracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
		err := step.Cleanup(e.t, e.K8SClient)
		Untracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
		assert.NoError(e.t, err)
	}
}
