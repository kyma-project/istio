package clusterconfig_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type hyperscalerClientMock struct {
	isAws bool
}

func (hc *hyperscalerClientMock) IsAws() bool {
	return hc.isAws
}

var _ = Describe("Hyperscaler", func() {
	Context("IsHyperscalerAWS", func() {
		It("should be true if hyperscaler is aws", func() {
			// given
			hc := &hyperscalerClientMock{isAws: true}
			// when
			isAws := hc.IsAws()

			// then
			Expect(isAws).To(BeTrue())
		})

		It("should be false if hyperscaler is not aws", func() {
			// given
			hc := &hyperscalerClientMock{isAws: false}

			// when
			isAws := hc.IsAws()

			// then
			Expect(isAws).To(BeFalse())
		})

	})
})
