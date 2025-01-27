package istio_test

import (
	"fmt"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	mockIstioTag             string = "1.16.1-distroless"
	lastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
)

var _ = Describe("Istio Configuration", func() {
	Context("LastAppliedConfiguration", func() {
		It("should update lastAppliedConfiguration and is able to unmarshal it back from annotation", func() {
			// given
			numTrustedProxies := 1
			istioCR := operatorv1alpha2.Istio{Spec: operatorv1alpha2.IstioSpec{Config: operatorv1alpha2.Config{NumTrustedProxies: &numTrustedProxies}}}

			// when
			err := istio.UpdateLastAppliedConfiguration(&istioCR, mockIstioTag)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(istioCR.Annotations).To(Not(BeEmpty()))
			Expect(istioCR.Annotations[lastAppliedConfiguration]).To(Equal(fmt.Sprintf(`{"config":{"numTrustedProxies":1,"telemetry":{"metrics":{}}},"IstioTag":"%s"}`, mockIstioTag)))

			appliedConfig, err := istio.GetLastAppliedConfiguration(&istioCR)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(*appliedConfig.Config.NumTrustedProxies).To(Equal(1))
		})
	})
})
