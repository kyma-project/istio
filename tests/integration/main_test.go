package integration

import (
	"context"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	iop "istio.io/istio/operator/pkg/apis"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	evaluationEnv string = "TEST_EVALUATION"

	productionMainSuitePath    string = "features/istio/production/main-suite"
	productionUpgradeSuitePath string = "features/istio/production/upgrade-suite"
	evaluationPath             string = "features/istio/evaluation"
)

func TestIstioMain(t *testing.T) {
	suiteName := "Istio Install"
	featurePath := productionMainSuitePath
	ev, ok := os.LookupEnv(evaluationEnv)
	if ok {
		if ev == "TRUE" {
			featurePath = evaluationPath
		}
	}

	runTestSuite(t, initScenario, featurePath, suiteName)
}

func TestIstioUpgrade(t *testing.T) {
	suiteName := "Istio Upgrade"
	runTestSuite(t, upgradeInitScenario, productionUpgradeSuitePath, suiteName)
}

func runTestSuite(t *testing.T, scenarioInit func(ctx *godog.ScenarioContext), featurePath string, suiteName string) {
	goDogOpts := godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "pretty",
		Paths:  []string{featurePath},
		// Concurrency must be set to 1, as the tests modify the global cluster state and can't be isolated.
		Concurrency: 1,
		// We want to randomize the scenario order to avoid any implicit dependencies between scenarios.
		Randomize:      time.Now().UTC().UnixNano(),
		DefaultContext: createDefaultContext(t),
		Strict:         true,
	}
	if shouldExportResults() {
		goDogOpts.Format = "pretty,junit:junit-report.xml,cucumber:cucumber-report.json"
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
	if shouldExportResults() {
		err := generateReport(suiteName)
		if err != nil {
			t.Errorf("error while generating report: %s", err)
		}
	}
}

func shouldExportResults() bool {
	return os.Getenv("EXPORT_RESULT") == "true"
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

	err = iop.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	err = istioCR.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	err = v1beta1.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	err = v1alpha3.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	return c
}

// test
