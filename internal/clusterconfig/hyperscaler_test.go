package clusterconfig_test

import (
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type awsClientMock struct {
	isAws bool
}

func (ac *awsClientMock) IsAws() bool {
	return ac.isAws
}

var _ = Describe("Hyperscaler", func() {
	Context("IsHyperscalerAWS", func() {
		It("should be true if hyperscaler is aws", func() {
			// given
			ac := &awsClientMock{isAws: true}
			// when
			isAws := clusterconfig.IsHyperscalerAWS(ac)

			// then
			Expect(isAws).To(BeTrue())
		})

		It("should be false if hyperscaler is not aws", func() {
			// given
			ac := &awsClientMock{isAws: false}

			// when
			isAws := clusterconfig.IsHyperscalerAWS(ac)

			// then
			Expect(isAws).To(BeFalse())
		})

	})
})
