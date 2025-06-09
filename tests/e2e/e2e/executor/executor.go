package executor

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"testing"
)

type Executor struct {
	LogOutputs   []*log.Logger
	TraceOutputs []*log.Logger
	DebugOutput  *log.Logger

	Ctx       context.Context
	K8SClient client.Client

	steps []Step

	t *testing.T
}

const (
	defaultLogDirectory = "logs"
	defaultTracePrefix  = "[TRACE] "
	defaultLogPrefix    = "[LOG] "
	defaultDebugPrefix  = "[DEBUG] "
)

func DefaultExecutor(t *testing.T, logDirectory string) *Executor {
	// Create a default log directory if it doesn't exist
	writeDirectory := defaultLogDirectory
	if logDirectory != "" {
		writeDirectory = fmt.Sprintf("%s/%s", logDirectory, defaultLogDirectory)
	}
	if err := os.MkdirAll(writeDirectory, 0755); err != nil {
		t.Fatalf("Failed to create log directory: %s", err.Error())
	}

	// Set up log files
	logFile, err := os.OpenFile(fmt.Sprintf("%s/log.log", writeDirectory),
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open log file: %s", err.Error())
	}

	traceFile, err := os.OpenFile(fmt.Sprintf("%s/trace.log", writeDirectory),
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open trace file: %s", err.Error())
	}

	debugFile, err := os.OpenFile(fmt.Sprintf("%s/debug.log", writeDirectory),
		os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open k8s debug file: %s", err.Error())
	}

	return NewExecutor(
		t,
		[]*log.Logger{
			log.New(log.Writer(), defaultLogPrefix, log.LstdFlags),
			log.New(logFile, defaultLogPrefix, log.LstdFlags),
			log.New(debugFile, defaultLogPrefix, log.LstdFlags),
		},
		[]*log.Logger{
			log.New(traceFile, defaultTracePrefix, log.LstdFlags),
			log.New(debugFile, defaultTracePrefix, log.LstdFlags),
		},
		log.New(debugFile, defaultDebugPrefix, log.LstdFlags),
	)
}

func NewExecutor(t *testing.T, logOutputs []*log.Logger,
	traceOutputs []*log.Logger, debugOutput *log.Logger) *Executor {
	k8sClient, err := setup.ClientFromKubeconfig(debugOutput)
	if err != nil {
		t.Fatalf("Failed to create Kubernetes client: %s", err.Error())
	}

	return &Executor{
		LogOutputs:   logOutputs,
		TraceOutputs: traceOutputs,
		t:            t,
		K8SClient:    k8sClient,
		DebugOutput:  debugOutput,
		Ctx:          context.Background(),
	}
}

func (e *Executor) Error(err error, args ...interface{}) {
	for _, logOutput := range e.LogOutputs {
		writeToOutput(logOutput, fmt.Sprintf("Error: %s", err.Error()), args...)
	}
	e.t.Fail()
}

func (e *Executor) Log(message string) {
	for _, logOutput := range e.LogOutputs {
		logOutput.Print(message)
	}
}

func writeToOutput(output *log.Logger, message string, args ...interface{}) {
	msg := strings.Builder{}
	msg.WriteString(fmt.Sprintf("%s", message))

	for _, arg := range args {
		msg.WriteString(fmt.Sprintf("%v", arg))
	}
	output.Print(msg.String())
}

func (e *Executor) trace(message string) {
	for _, traceOutput := range e.TraceOutputs {
		writeToOutput(traceOutput, "Begin: ", message)
	}
}

func (e *Executor) untrace(message string) {
	for _, traceOutput := range e.TraceOutputs {
		writeToOutput(traceOutput, "End: ", message)
	}
}

func (e *Executor) RunStep(step Step) error {
	e.trace(step.Description())

	if err := step.Execute(e.Ctx, e.K8SClient, e.DebugOutput); err != nil {
		e.Error(err)
		return fmt.Errorf("failed to execute step %s: %w", step.Description(), err)
	}

	e.untrace(step.Description())

	e.steps = append(e.steps, step)
	return nil
}

func (e *Executor) Cleanup() {
	for _, step := range e.steps {
		e.trace(fmt.Sprintf("Cleaning up step: %s", step.Description()))
		err := step.Cleanup(e.Ctx, e.K8SClient)
		e.untrace(fmt.Sprintf("Cleaning up step: %s", step.Description()))
		assert.NoError(e.t, err)
	}
}
