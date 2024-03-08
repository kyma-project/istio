package described_errors_test

import (
	"github.com/kyma-project/istio/operator/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"testing"
)

func TestErrors(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Described Errors Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("described-errors-suite", report)
})
