package exec

import (
	"context"
	"errors"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"os/exec"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
	"syscall"
	"testing"
)

type Command struct {
	Command string

	Output   []byte
	ExitCode int

	CleanupCmd string
}

func (b *Command) Description() string {
	return fmt.Sprintf("Executing command: %s", b.Command)
}

func (b *Command) Execute(t *testing.T, _ context.Context, _ client.Client) error {
	splitCommand := strings.Split(b.Command, " ")
	cmd := exec.Command(splitCommand[0], splitCommand[1:]...)
	executor.Debugf(t, "Executing command: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				b.ExitCode = status.ExitStatus()
			} else {
				b.ExitCode = -1
			}
		}
		return fmt.Errorf("recieved err=%w; Output=%s", err, string(output))
	}
	b.Output = output
	b.ExitCode = 0
	executor.Debugf(t, "Command output:\n%s", string(b.Output))
	return nil
}

func (b *Command) Cleanup(t *testing.T, _ context.Context, _ client.Client) error {
	if b.CleanupCmd == "" {
		executor.Debugf(t, "No cleanup command specified, skipping cleanup")
		return nil
	}

	splitCommand := strings.Split(b.CleanupCmd, " ")
	cmd := exec.Command(splitCommand[0], splitCommand[1:]...)
	executor.Debugf(t, "Executing cleanup command: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	executor.Debugf(t, "Cleanup command output:\n%s", string(output))
	return nil
}
