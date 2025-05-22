package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/kyma-project/istio/operator/tests/testcontext"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	securityv1 "istio.io/client-go/pkg/apis/security/v1"
	telemetryv1 "istio.io/client-go/pkg/apis/telemetry/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	istiov1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func TestEvaluation(t *testing.T) {
	suiteName := "Evaluation"
	runTestSuite(t, initScenario, "features/evaluation-suite", suiteName)
}

func TestConfiguration(t *testing.T) {
	suiteName := "Configuration"
	runTestSuite(t, initScenario, "features/configuration-suite", suiteName)
}
func TestMeshCommunication(t *testing.T) {
	suiteName := "Mesh Communication"
	runTestSuite(t, initScenario, "features/mesh-communication-suite", suiteName)
}
func TestInstallation(t *testing.T) {
	suiteName := "Installation"
	runTestSuite(t, initScenario, "features/installation-suite", suiteName)
}
func TestObservability(t *testing.T) {
	suiteName := "Observability"
	runTestSuite(t, initScenario, "features/observability-suite", suiteName)
}

func TestUpgrade(t *testing.T) {
	suiteName := "Upgrade"
	runTestSuite(t, initScenario, "features/upgrade-suite", suiteName)
}

func TestAws(t *testing.T) {
	suiteName := "AWS Specific"
	runTestSuite(t, initScenario, "features/aws-suite", suiteName)
}

func TestGcp(t *testing.T) {
	suiteName := "GCP Specific"
	runTestSuite(t, initScenario, "features/gcp-suite", suiteName)
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
		TestingT:       t,
		StopOnFailure:  true,
	}
	if shouldExportResults() {
		goDogOpts.Format = "pretty,cucumber:cucumber-report.json"
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
	ctx := testcontext.SetK8sClientInContext(t.Context(), createK8sClient())
	return testcontext.SetTestingInContext(ctx, t)
}

func createK8sClient() client.Client {
	c, err := client.New(config.GetConfigOrDie(), client.Options{})
	if err != nil {
		panic(err)
	}

	err = istiov1alpha2.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	err = networkingv1.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	err = securityv1.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	err = telemetryv1.AddToScheme(c.Scheme())
	if err != nil {
		panic(err)
	}

	return c
}
