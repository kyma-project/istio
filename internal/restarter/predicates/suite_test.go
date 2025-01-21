package predicates_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
)

func TestRestarter(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Restarter Predicates Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("restarter-predicates-suite", report)
})
