package integration

import (
	"context"
	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	iop "istio.io/istio/operator/pkg/apis"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"testing"
	"time"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func TestIstio(t *testing.T) {

	goDogOpts := godog.Options{
		Output: colors.Colored(os.Stdout),
		Format: "pretty",
		Paths:  []string{"features/istio"},
		// Concurrency must be set to 1, as the tests modify the global cluster state and can't be isolated.
		Concurrency: 1,
		// We want to randomize the scenario order to avoid any implicit dependencies between scenarios.
		Randomize:      time.Now().UTC().UnixNano(),
		DefaultContext: createDefaultContext(t),
	}

	if os.Getenv("EXPORT_RESULT") == "true" {
		goDogOpts.Format = "pretty,junit:junit-report.xml,cucumber:cucumber-report.json"
	}

	suite := godog.TestSuite{
		Name:                "istio",
		ScenarioInitializer: initScenario,
		Options:             &goDogOpts,
	}
	testExitCode := suite.Run()

	if os.Getenv("EXPORT_RESULT") == "true" {
		err := generateReport("istio-installation")
		if err != nil {
			t.Errorf("error while generating report: %s", err)
		}
	}

	if testExitCode != 0 {
		t.Fatalf("non-zero status returned, failed to run feature tests")
	}
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
	return c
}
