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
	Cleanup(context.Context, client.Client) error
}

type Executor struct {
	t     *testing.T
	steps []Step
	isCi  bool

	K8SClient client.Client
}

func NewExecutor(t *testing.T) *Executor {
	k8sClient, err := setup.ClientFromKubeconfig(t)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %s", err.Error())
	}

	ciEnv := os.Getenv("CI")

	return &Executor{
		t:         t,
		K8SClient: k8sClient,
		isCi:      ciEnv == "true",
	}
}

const (
	ErrorPrefix   = "[ERROR] "
	InfoPrefix    = "[INFO] "
	TracePrefix   = "[TRACE] "
	UntracePrefix = "[UNTRACE] "
	DebugPrefix   = "[DEBUG] "
)

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
	Tracef(e.t, step.Description())

	if err := step.Execute(e.t, e.t.Context(), e.K8SClient); err != nil {
		return fmt.Errorf("failed to execute step %s: %w", step.Description(), err)
	}

	Untracef(e.t, step.Description())

	e.steps = append(e.steps, step)
	return nil
}

func (e *Executor) Cleanup() {
	if e.isCi {
		return
	}

	for _, step := range e.steps {
		Tracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
		err := step.Cleanup(e.t.Context(), e.K8SClient)
		Untracef(e.t, fmt.Sprintf("Cleaning up step: %s", step.Description()))
		assert.NoError(e.t, err)
	}
}
