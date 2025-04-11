package test

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
		Name:                "ProxyRestartTestSuite",
		ScenarioInitializer: InitializeScenario,
		Options:             &prodOpts,
	}

	suiteExitCode := suite.Run()

	if os.Getenv(exportResultVar) == "true" {
		generateReport()
	}

	if suiteExitCode != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}

func generateReport() {
	htmlOutputDir := "reports/"

	htmlReport := gocure.HTML{
		Config: html.Data{
			InputJsonPath:    "cucumber-report.json",
			OutputHtmlFolder: htmlOutputDir,
			Title:            "Kyma Istio Restart tests",
			Metadata: models.Metadata{
				Platform:   runtime.GOOS,
				Parallel:   "Scenarios",
				Executed:   "Remote",
				AppVersion: "main",
				Browser:    "default",
			},
		},
	}
	err := htmlReport.Generate()
	if err != nil {
		log.Fatal(err.Error())
	}

	err = filepath.Walk("reports", func(path string, info fs.FileInfo, err error) error {
		if path == "reports" {
			return nil
		}

		data, err1 := os.ReadFile(path)
		if err1 != nil {
			return err
		}

		//Format all patterns like "&lt" to not be replaced later
		find := regexp.MustCompile(`&\w\w`)
		formatted := find.ReplaceAllFunc(data, func(b []byte) []byte {
			return []byte{b[0], ' ', b[1], b[2]}
		})

		err = os.WriteFile(path, formatted, fs.FileMode(02))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		log.Fatal(err.Error())
	}

	if artifactsDir, ok := os.LookupEnv("ARTIFACTS"); ok {
		err = filepath.Walk("reports", func(path string, info fs.FileInfo, err error) error {
			if path == "reports" {
				return nil
			}

			_, err1 := copyReport(path, fmt.Sprintf("%s/report.html", artifactsDir))
			if err1 != nil {
				return err1
			}
			return nil
		})

		if err != nil {
			log.Fatal(err.Error())
		}

		_, err = copyReport("./junit-report.xml", fmt.Sprintf("%s/junit-report.xml", artifactsDir))
		if err != nil {
			log.Fatal(err.Error())
		}
	}

}

func copyReport(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer func() {
		e := source.Close()
		if e != nil {
			log.Printf("error closing source file: %s", e.Error())
		}
	}()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer func() {
		e := destination.Close()
		if e != nil {
			log.Printf("error closing destination file: %s", e.Error())
		}
	}()

	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}
