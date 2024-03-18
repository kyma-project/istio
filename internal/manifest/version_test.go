package manifest

import (
	"fmt"

	"github.com/coreos/go-semver/semver"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetModuleVersion", func() {
	It("should return version from package variable", func() {
		Expect(GetModuleVersion()).To(Equal(version))
	})
})

var _ = Describe("GetIstioVersion", func() {
	merger := NewDefaultIstioMerger()

	It("should return Istio version from tag in production manifest file", func() {
		// given

		// when
		version, prerelease, err := GetIstioVersion(&merger)
		Expect(err).Should(Not(HaveOccurred()))

		iop, err := merger.GetIstioOperator(clusterconfig.Production)
		Expect(err).Should(Not(HaveOccurred()))

		prodVersion, err := semver.NewVersion(iop.Spec.Tag.GetStringValue())
		Expect(err).Should(Not(HaveOccurred()))

		// then
		Expect(version).To(Not(BeEmpty()))
		Expect(prerelease).To(Not(BeEmpty()))
		Expect(version).To(Equal(fmt.Sprintf("%d.%d.%d", prodVersion.Major, prodVersion.Minor, prodVersion.Patch)))
		Expect(prerelease).To(Equal(string(prodVersion.PreRelease)))
	})

	It("should have same version in evaluation and production manifest files", func() {
		// given

		// when
		prodIOP, err := merger.GetIstioOperator(clusterconfig.Production)
		Expect(err).Should(Not(HaveOccurred()))

		prodVersion, err := semver.NewVersion(prodIOP.Spec.Tag.GetStringValue())
		Expect(err).Should(Not(HaveOccurred()))

		evalIOP, err := merger.GetIstioOperator(clusterconfig.Evaluation)
		Expect(err).Should(Not(HaveOccurred()))

		evalVersion, err := semver.NewVersion(evalIOP.Spec.Tag.GetStringValue())
		Expect(err).Should(Not(HaveOccurred()))

		// then
		Expect(prodVersion.Equal(*evalVersion)).To(BeTrue())
	})
})
