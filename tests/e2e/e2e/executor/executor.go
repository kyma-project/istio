package executor

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"github.com/stretchr/testify/assert"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

type Step interface {
	Description() string
	Execute(*testing.T, context.Context, client.Client) error
	Cleanup(*testing.T, context.Context, client.Client) error
}

type Executor struct {
	t     *testing.T
	steps []Step

	// isCi indicates whether the tests are running in a CI environment
	// i.e., if the environment variable CI is set to "true"
	// This is used to skip cleanup steps in CI environments.
	isCi bool

	// OnlyCleanup indicates whether to skip step execution and only perform cleanup
	onlyCleanup bool

	K8SClient client.Client
}

func NewExecutor(t *testing.T) *Executor {
	k8sClient, err := setup.ClientFromKubeconfig(t)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %s", err.Error())
	}

	ciEnv := os.Getenv("CI")
	onlyCleanup := os.Getenv("ONLY_CLEANUP")

	return &Executor{
		t:           t,
		K8SClient:   k8sClient,
		isCi:        ciEnv == "true",
		onlyCleanup: onlyCleanup == "true",
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
	e.steps = append(e.steps, step)
	if e.onlyCleanup {
		Debugf(e.t, "Skipping step execution in cleanup mode: %s", step.Description())
		return nil
	}

	Tracef(e.t, step.Description())

	if err := step.Execute(e.t, e.t.Context(), e.K8SClient); err != nil {
		Errorf(e.t, "Failed to execute step: %s err=%s", step.Description(), err.Error())
		return err
	}

	Untracef(e.t, step.Description())
	return nil
}

func (e *Executor) Cleanup() {
	if e.isCi && !e.onlyCleanup {
		Infof(e.t, "Skipping cleanup in CI environment")
		return
	}

	Tracef(e.t, "Starting cleanup")
	defer Untracef(e.t, "Finished cleanup")

	// Perform cleanup in reverse order
	for i := len(e.steps) - 1; i >= 0; i-- {
		step := e.steps[i]
		Tracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
		err := step.Cleanup(e.t, e.t.Context(), e.K8SClient)
		Untracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
		assert.NoError(e.t, err)
	}
}
