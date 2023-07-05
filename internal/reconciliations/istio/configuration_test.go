package istio_test

import (
	"fmt"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	mockIstioTag             string = "1.16.1-distroless"
	lastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
)

var _ = Describe("CR configuration", func() {
	Context("UpdateLastAppliedConfiguration", func() {
		It("should update CR with IstioVersion and spec of CR", func() {
			// given
			cr := operatorv1alpha1.Istio{}

			// when
			updatedCR, err := istio.UpdateLastAppliedConfiguration(cr, mockIstioTag)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(updatedCR.Annotations).To(Not(BeEmpty()))

			Expect(updatedCR.Annotations[lastAppliedConfiguration]).To(Equal(fmt.Sprintf(`{"config":{},"IstioTag":"%s"}`, mockIstioTag)))
		})
	})
})
