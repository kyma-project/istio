package istio_resources_test

import (
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Istio resources", func() {
	Context("Get", func() {
		It("Resources when hyperscaler is AWS", func() {
			_, istioResources := istio_resources.Get()
			found := false
			for _, ir := range istioResources {
				if ir.Name() == "EnvoyFilter/proxy-protocol" {
					found = true
				}
			}
			Expect(found).To(BeTrue())

		})
		It("Resources when hyperscaler is not AWS", func() {})
	})
})
