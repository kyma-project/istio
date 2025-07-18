package testsupport

import (
	"fmt"
	"os"
)

type EnvVar struct {
	Name    string
	Default string
}

var requiredEnvs = []EnvVar{
	{"OPERATOR_VERSION", "dev"},
}

func EnvironmentVariables() error {
	ciMode := os.Getenv("GITHUB_WORKFLOW")
	if ciMode != "true" {
		err := setDefaultEnvs()
		if err != nil {
			return err
		}
	}

	err := validateEnvs()
	if err != nil {
		return err
	}

	return nil
}

func GetOperatorVersion() string {
	return os.Getenv("OPERATOR_VERSION")
}

func setDefaultEnvs() error {
	for _, env := range requiredEnvs {
		err := os.Setenv(env.Name, env.Default)
		if err != nil {
			return fmt.Errorf("failed to set default environment variable: %s with error: %s", env.Name, err.Error())
		}
	}

	return nil
}

func validateEnvs() error {
	for _, env := range requiredEnvs {
		if os.Getenv(env.Name) == "" {
			return fmt.Errorf("required environment variable is not set: %s. Please set it to run the tests", env.Name)
		}
	}

	return nil
}
