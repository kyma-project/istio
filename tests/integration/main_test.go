package integration

import (
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/spf13/pflag"
	iop "istio.io/istio/operator/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"

	"github.com/avast/retry-go"
	"github.com/cucumber/godog"
	"log"
	"os"
	"testing"

	"github.com/vrischmann/envconfig"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func TestIstioJwt(t *testing.T) {
	InitTestSuite()

	opts := goDogOpts
	opts.Paths = []string{"features/istio"}
	opts.Concurrency = conf.TestConcurrency

	test := TestWithTemplatedManifest{}

	suite := godog.TestSuite{
		Name: "istio",
		// We are not using ScenarioInitializer, as this function only needs to set up global resources
		TestSuiteInitializer: func(ctx *godog.TestSuiteContext) {
			test.initIstioScenarios(ctx.ScenarioContext())
		},
		Options: &opts,
	}

	testExitCode := suite.Run()
	if testExitCode != 0 {
		t.Fatalf("non-zero status returned, failed to run feature tests")
	}
}

func InitTestSuite() {
	pflag.Parse()
	goDogOpts.Paths = pflag.Args()

	if os.Getenv(exportResultVar) == "true" {
		goDogOpts.Format = "pretty,junit:junit-report.xml,cucumber:cucumber-report.json"
	}

	if err := envconfig.Init(&conf); err != nil {
		log.Fatalf("Unable to setup config: %v", err)
	}

	retryOpts = []retry.Option{
		retry.Delay(conf.ReqDelay),
		retry.Attempts(uint(conf.ReqTimeout / conf.ReqDelay)),
		retry.DelayType(retry.FixedDelay),
	}

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
	k8sClient = c
}
