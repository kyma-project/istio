package testsupport

import (
	"fmt"
	"log"
	"os"
)

type EnvVar struct {
	Name    string
	Default string
}

var requiredEnvs = []EnvVar{
	{"OPERATOR_VERSION", "dev"},
}

func LoadEnvironmentVariables() error {
	// env variable should always be set to true in the workflow
	// according to the documentation: https://docs.github.com/en/actions/reference/variables-reference#default-environment-variables
	// it is present in the Github Actions workflows
	ciMode := os.Getenv("CI")
	if ciMode != "true" {
		log.Println("CI environment variable is not set to true, setting default environment variables")
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
