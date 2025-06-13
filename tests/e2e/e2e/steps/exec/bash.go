package exec

import (
	"errors"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	"os/exec"
	"strings"
	"syscall"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Command struct {
	Command string

	output   []byte
	exitCode int

	CleanupCmd string
}

func (c *Command) Output() ([]byte, int) {
	return c.output, c.exitCode
}

func (c *Command) Description() string {
	return fmt.Sprintf("Executing command: %s", c.Command)
}

func (c *Command) Execute(t *testing.T, _ client.Client) error {
	splitCommand := strings.Split(c.Command, " ")
	cmd := exec.Command(splitCommand[0], splitCommand[1:]...) // #nosec G204
	logging.Debugf(t, "Executing command: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				c.exitCode = status.ExitStatus()
			} else {
				c.exitCode = -1
			}
		}
		return fmt.Errorf("recieved err=%w; Output=%s", err, string(output))
	}

	logging.Debugf(t, "Command output:\n%s", string(output))
	c.output = output
	c.exitCode = 0

	return nil
}

func (c *Command) Cleanup(t *testing.T, _ client.Client) error {
	if c.CleanupCmd == "" {
		logging.Debugf(t, "No cleanup command specified, skipping cleanup")
		return nil
	}

	splitCommand := strings.Split(c.CleanupCmd, " ")
	cmd := exec.Command(splitCommand[0], splitCommand[1:]...) // #nosec G204
	logging.Debugf(t, "Executing cleanup command: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	logging.Debugf(t, "Cleanup command output:\n%s", string(output))
	return nil
}
