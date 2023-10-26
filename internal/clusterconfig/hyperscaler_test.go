package clusterconfig_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Hyperscaler", func() {
	Context("IsHyperscalerAWS", func() {
		It("should be true if hyperscaler is AWS", func() {
			isAws, err := clusterconfig.IsHyperscalerAWS()
			Expect(err).ToNot(HaveOccurred())
			Expect(isAws).To(BeTrue())
		})

	})
})
