package external_installation_test

import (
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/external_installation"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"testing"
)

func TestManifest(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Test External Installer")
}

var _ = Describe("External Install Client", func() {
	Context("Create external install client", func() {
		It("should create external installer with correct istioOperator path", func() {
			iopPath := "/some/path"
			compatibilityMode := true
			istioVersion := "1.21.2"
			ei, err := external_installation.NewExternalInstaller(iopPath, istioVersion, compatibilityMode)
			Expect(err).ToNot(HaveOccurred())
			Expect(ei.Args[1]).To(Equal(iopPath))
		})

		It("should create external installer with compatibility flag set when compatibilityMode is on", func() {
			iopPath := "/some/path"
			compatibilityMode := true
			istioVersion := "1.21.2"
			ei, err := external_installation.NewExternalInstaller(iopPath, istioVersion, compatibilityMode)
			Expect(err).ToNot(HaveOccurred())
			Expect(ei.Args[2]).To(Equal("compatibilityVersion=1.20"))
		})

		It("should create external installer without compatibility flag set when compatibilityMode is off", func() {
			iopPath := "/some/path"
			compatibilityMode := false
			istioVersion := "1.21.2"
			ei, err := external_installation.NewExternalInstaller(iopPath, istioVersion, compatibilityMode)
			Expect(err).ToNot(HaveOccurred())
			Expect(ei.Args[2]).To(Equal(""))
		})

		It("should fail if istio version is not following the semver", func() {
			iopPath := "/some/path"
			compatibilityMode := true
			istioVersion := "1.21"
			_, err := external_installation.NewExternalInstaller(iopPath, istioVersion, compatibilityMode)
			Expect(err).To(HaveOccurred())
		})
	})
})
