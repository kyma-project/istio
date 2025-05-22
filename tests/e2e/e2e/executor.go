package e2e

import (
	"fmt"
	"github.com/cucumber/godog/colors"
	"log"
	"strings"
	"testing"
)

type Executor struct {
	Steps        []Step
	LogOutputs   []*log.Logger
	TraceOutputs []*log.Logger

	t *testing.T
}

func NewExecutor(t *testing.T, steps []Step, logOutputs []*log.Logger, traceOutputs []*log.Logger) *Executor {
	return &Executor{
		Steps:        steps,
		LogOutputs:   logOutputs,
		TraceOutputs: traceOutputs,
		t:            t,
	}
}

func (e *Executor) Error(err error, args ...interface{}) {
	for _, logOutput := range e.LogOutputs {
		writeToOutput(logOutput, colors.Red(fmt.Sprintf("Error: %s", err.Error())), args...)
	}
	e.t.Fail()
}

func (e *Executor) Log(message string, args ...interface{}) {
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

func (e *Executor) Trace(message string, args ...interface{}) error {
	for _, traceOutput := range e.TraceOutputs {
		writeToOutput(traceOutput, fmt.Sprintf("S:%s", message), args...)
	}
	return nil
}

func (e *Executor) Untrace(message string, args ...interface{}) error {
	for _, traceOutput := range e.TraceOutputs {
		writeToOutput(traceOutput, "E:", message, args)
	}
	return nil
}

func (e *Executor) Execute() error {
	for _, step := range e.Steps {
		err := e.Trace(step.Name(), step.Args())
		if err != nil {
			e.Error(err)
		}

		if err := step.Execute(); err != nil {
			e.Error(err)
		}
		if err := step.AssertSuccess(); err != nil {
			return fmt.Errorf("step %s assertion failed: %w", step.Name(), err)
		}

		//e.Debug(step.Name(), step.Args(), "Output:", step.Output()); err != nil {
		err = e.Untrace(step.Name(), step.Args())
	}
	return nil
}
