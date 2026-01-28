package config

import "os"

type Config struct {
	OperatorVersion string
	SkipCleanup bool
}

func Get() *Config {
	return &Config{
		OperatorVersion: getEnvOrDefault("OPERATOR_VERSION", "dev"),
		SkipCleanup:     getEnvAsBool("SKIP_CLEANUP", false),
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value == "true" || value == "1" || value == "yes"
}
