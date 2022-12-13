package test

import (
	"os"
	"testing"

	_ "embed"
	"log"
	"runtime"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"

	"gitlab.com/rodrigoodhin/gocure/models"
	"gitlab.com/rodrigoodhin/gocure/pkg/gocure"
	"gitlab.com/rodrigoodhin/gocure/report/html"
)

var t *testing.T

const exportResultVar = "EXPORT_RESULTS"

var goDogOpts = godog.Options{
	Output:      colors.Colored(os.Stdout),
	Format:      "pretty",
	TestingT:    t,
	Concurrency: 1,
}

func InitTest() {
	if os.Getenv(exportResultVar) == "true" {
		goDogOpts.Format = "pretty,junit:junit-report.xml,cucumber:cucumber-report.json"
	}
}

func TestProxyReset(t *testing.T) {
	InitTest()

	prodOpts := goDogOpts
	prodOpts.Paths = []string{"features/"}

	suite := godog.TestSuite{
		Name:                "ProxyResetTestSuite",
		ScenarioInitializer: InitializeScenario,
		Options:             &prodOpts,
	}

	suiteExitCode := suite.Run()

	if os.Getenv(exportResultVar) == "true" {
		generateHTMLReport()
	}

	if suiteExitCode != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func generateHTMLReport() {
	html := gocure.HTML{
		Config: html.Data{
			InputJsonPath:    "cucumber-report.json",
			OutputHtmlFolder: "reports/",
			Title:            "Kyma Istio component tests",
			Metadata: models.Metadata{
				TestEnvironment: "fake",
				Platform:        runtime.GOOS,
				Parallel:        "Scenarios",
				Executed:        "Remote",
				AppVersion:      "main",
				Browser:         "default",
			},
		},
	}
	err := html.Generate()
	if err != nil {
		log.Fatalf(err.Error())
	}
}
