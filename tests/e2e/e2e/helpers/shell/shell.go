package shell

import (
	"os/exec"
	"testing"
)

func Execute(t *testing.T, name string, arg ...string) ([]byte, error) {
	t.Helper()
	t.Logf("Executing shell command %s with args %v", name, arg)
	cmd := exec.Command(name, arg...)
	output, err := cmd.CombinedOutput()
	t.Logf("Output: %s", string(output))
	return output, err
}
