package istio_resources_test

import (
	"testing"

	"github.com/onsi/ginkgo/v2/types"

	"github.com/kyma-project/istio/operator/internal/tests"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Istio resources Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("istio-resources-suite", report)
})
