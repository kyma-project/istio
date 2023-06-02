package integration

import (
	"context"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	iop "istio.io/istio/operator/pkg/apis"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func TestIstio(t *testing.T) {
	defaultContext := testcontext.SetK8sClientInContext(context.Background(), createK8sClient())

	goDogOpts := godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "pretty",
		Paths:  []string{"features/istio"},
		// Concurrency must be set to 1, as the tests modify the global cluster state and can't be isolated.
		Concurrency:    1,
		DefaultContext: defaultContext,
	}

	if os.Getenv("EXPORT_RESULT") == "true" {
		goDogOpts.Format = "pretty,junit:junit-report.xml,cucumber:cucumber-report.json"
	}

	suite := godog.TestSuite{
		Name: "istio",
		// We are not using ScenarioInitializer, as this function only needs to set up global resources
		TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
			initScenarios(ctx.ScenarioContext())
		},
		Options: &goDogOpts,
	}

	testExitCode := suite.Run()
	if testExitCode != 0 {
		t.Fatalf("non-zero status returned, failed to run feature tests")
	}
}

func createK8sClient() client.Client {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		panic(err)
	}

	err = iop.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	err = istioCR.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}
	return c
}
