package restarter_test

import (
	"github.com/kyma-project/istio/operator/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"testing"
)

func TestRestarter(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Restarter Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("restarter-suite", report)
})
