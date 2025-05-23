package performance_grpc

import (
	"github.com/cucumber/godog"

	"github.com/kyma-project/istio/operator/tests/performance-grpc/steps"
)

func initScenario(ctx *godog.ScenarioContext) {
	t := steps.TemplatedPerformanceJob{}

	ctx.Step(`^"([^"]*)" is set to "([^"]*)"$`, t.SetTemplateValue)
	ctx.Step(`^the gRPC performance test is executed$`, t.ExecutePerformanceTest)
	ctx.Step(`^the test should run successfully$`, t.TestShouldRunSuccessfully)
}
