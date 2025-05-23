package configuration_test

import (
	"fmt"
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/onsi/ginkgo/v2/types"

	"github.com/kyma-project/istio/operator/internal/tests"
)

const (
	mockIstioTag             string = "1.16.1-distroless"
	lastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
)

func TestRestarter(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Istio Configuration Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("istio-configuration-suite", report)
})

var _ = Describe("Istio Configuration", func() {
	Context("LastAppliedConfiguration", func() {
		It("should update lastAppliedConfiguration and is able to unmarshal it back from annotation", func() {
			// given
			numTrustedProxies := 1
			istioCR := operatorv1alpha2.Istio{Spec: operatorv1alpha2.IstioSpec{Config: operatorv1alpha2.Config{NumTrustedProxies: &numTrustedProxies}}}

			// when
			err := configuration.UpdateLastAppliedConfiguration(&istioCR, mockIstioTag)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(istioCR.Annotations).To(Not(BeEmpty()))
			Expect(
				istioCR.Annotations[lastAppliedConfiguration],
			).To(Equal(fmt.Sprintf(`{"config":{"numTrustedProxies":1,"telemetry":{"metrics":{}}},"IstioTag":"%s"}`, mockIstioTag)))

			appliedConfig, err := configuration.GetLastAppliedConfiguration(&istioCR)

			Expect(err).ShouldNot(HaveOccurred())
			Expect(*appliedConfig.Config.NumTrustedProxies).To(Equal(1))
		})
	})

	Context("CheckIstioVersionUpdate", func() {
		It("should return nil when target version is the same as current version", func() {
			err := configuration.CheckIstioVersionUpdate("1.10.0", "1.10.0")
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should return nil when target version is one minor version higher than current version", func() {
			err := configuration.CheckIstioVersionUpdate("1.10.0", "1.11.0")
			Expect(err).ShouldNot(HaveOccurred())
		})

		It("should return error when target version is lower than current version", func() {
			err := configuration.CheckIstioVersionUpdate("1.11.0", "1.10.0")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("downgrade not supported"))
		})

		It("should return error when target version is more than one minor version higher than current version", func() {
			err := configuration.CheckIstioVersionUpdate("1.10.0", "1.12.0")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("the difference between versions exceed one minor version"))
		})

		It("should return error when target version has a different major version", func() {
			err := configuration.CheckIstioVersionUpdate("1.10.0", "2.10.0")
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("major version upgrade is not supported"))
		})

		It("should return nil when target version is a pre-release of the same version", func() {
			err := configuration.CheckIstioVersionUpdate("1.10.0", "1.10.0-beta.1")
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
