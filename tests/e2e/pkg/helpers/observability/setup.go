package observability

import (
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/telemetry"
)

// SetupLogs enables logs telemetry and returns error if it fails
func SetupLogs(t *testing.T) error {
	t.Helper()
	return telemetry.EnableLogs(t)
}

// SetupTraces enables traces telemetry and returns OtelCollectorInfo
func SetupTraces(t *testing.T) (*telemetry.OtelCollectorInfo, error) {
	t.Helper()

	// Enable traces telemetry
	err := telemetry.EnableTraces(t)
	if err != nil {
		return nil, err
	}

	// Create OTel mock collector
	return telemetry.CreateOtelMockCollector(t)
}
