package clusterconfig_test

import (
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hyperscaler", func() {
	Context("IsHyperscalerAWS", func() {
		It("should be true if hyperscaler is AWS", func() {
			// when
			isAws, err := clusterconfig.IsHyperscalerAWS()

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(isAws).To(BeTrue())
		})

		It("should be false if hyperscaler is not AWS", func() {
			// when
			isAws, err := clusterconfig.IsHyperscalerAWS()

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(isAws).To(BeFalse())
		})

	})
})
