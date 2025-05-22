package ingressgateway_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/istio/operator/internal/tests"
)

func TestRestarter(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Ingress Gateway Restarter Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("ingress-gateway-restarter", report)
})
