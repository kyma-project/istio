package steps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

func DeployControllerFromLocalSource(ctx context.Context) error {
	// Spawned process inherit env vars from the go test process.
	_, ok := os.LookupEnv("IMG")
	if !ok {
		return fmt.Errorf("provide IMG env variable to deploy new version of controller")
	}
	cmd := exec.CommandContext(ctx, "make", "deploy")
	// go test is invoked from tests/integration dir by Makefile
	// Set dir to root of the project to be able to invoke make deploy without additional parameters.
	cmd.Dir = "../.."
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
