package performance_grpc

import (
	"context"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/kyma-project/istio/operator/tests/testcontext"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
	"time"
)

const (
	grpcFeaturePath = "features"
)

func TestPerformanceGRPC(t *testing.T) {
	suiteName := "Performance GRPC"
	featurePath := grpcFeaturePath

	runTestSuite(t, initScenario, featurePath, suiteName)
}

func createDefaultContext(t *testing.T) context.Context {
	ctx := testcontext.SetK8sClientInContext(context.Background(), createK8sClient())
	return testcontext.SetTestingInContext(ctx, t)
}

func createK8sClient() client.Client {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		panic(err)
	}

	return c
}

func runTestSuite(t *testing.T, scenarioInit func(ctx *godog.ScenarioContext), featurePath string, suiteName string) {
	goDogOpts := godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "pretty",
		Paths:  []string{featurePath},
		// Concurrency must be set to 1, as the tests create a load on Istio ingress-gateway, which is a shared resource.
		Concurrency: 1,
		// We want to randomize the scenario to avoid results being affected by the order of execution.
		Randomize:      time.Now().UTC().UnixNano(),
		DefaultContext: createDefaultContext(t),
		Strict:         true,
		TestingT:       t,
	}

	suite := godog.TestSuite{
		Name:                suiteName,
		ScenarioInitializer: scenarioInit,
		Options:             &goDogOpts,
	}
	testExitCode := suite.Run()

	if testExitCode != 0 {
		t.Fatalf("non-zero status returned, failed to run feature tests")
	}
}
