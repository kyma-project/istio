package exec

import (
	"errors"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/logging"
	"os/exec"
	"syscall"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Command struct {
	Command string
	Args    []string

	Output   []byte
	ExitCode int

	CleanupCmd  string
	CleanupArgs []string
}

func (c *Command) Description() string {
	return fmt.Sprintf("Executing command: %s", c.Command)
}

func (c *Command) Execute(t *testing.T, _ client.Client) error {
	cmd := exec.Command(c.Command, c.Args...) // #nosec G204
	logging.Debugf(t, "Executing command: %s", cmd.String())

	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				c.ExitCode = status.ExitStatus()
			} else {
				c.ExitCode = -1
			}
		}
		return fmt.Errorf("recieved err=%w; Output=%s", err, string(output))
	}

	logging.Debugf(t, "Command output:\n%s", string(output))
	c.Output = output
	c.ExitCode = 0

	return nil
}

func (c *Command) Cleanup(t *testing.T, _ client.Client) error {
	if c.CleanupCmd == "" {
		logging.Debugf(t, "No cleanup command specified, skipping cleanup")
		return nil
	}

	cmd := exec.Command(c.Command, c.CleanupArgs...) // #nosec G204
	logging.Debugf(t, "Executing cleanup command: %s", cmd.String())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	logging.Debugf(t, "Cleanup command output:\n%s", string(output))
	return nil
}
