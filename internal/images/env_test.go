package images_test

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/istio/operator/internal/images"
)

func TestEnvs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Environment Suite")
}

var _ = Describe("Images.GetHubAndImageTag", func() {
	type fields struct {
		Pilot      images.Image
		InstallCNI images.Image
		ProxyV2    images.Image
		Ztunnel    images.Image
	}

	DescribeTable("GetHubAndImageTag",
		func(f fields, want images.HubTag, wantErr bool, expErr error) {
			e := &images.Images{
				Pilot:      f.Pilot,
				InstallCNI: f.InstallCNI,
				ProxyV2:    f.ProxyV2,
				Ztunnel:    f.Ztunnel,
			}
			got, err := e.GetHubAndImageTag()
			if wantErr {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("image"))
				Expect(err.Error()).To(ContainSubstring(expErr.Error()))
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(got).To(Equal(want))
			}
		},
		Entry("valid images",
			fields{
				Pilot:      "docker.io/istio/pilot:1.10.0",
				InstallCNI: "docker.io/istio/cni:1.10.0",
				ProxyV2:    "docker.io/istio/proxyv2:1.10.0",
				Ztunnel:    "docker.io/istio/ztunnel:1.10.0",
			},
			images.HubTag{Hub: "docker.io/istio", Tag: "1.10.0"},
			false,
			nil,
		),
		Entry("invalid image hub",
			fields{
				Pilot:      "pilot:1.10.0",
				InstallCNI: "docker.io/istio/cni:1.10.0",
				ProxyV2:    "docker.io/istio/proxyv2:1.10.0",
				Ztunnel:    "docker.io/istio/ztunnel:1.10.0",
			},
			images.HubTag{},
			true,
			fmt.Errorf("image pilot:1.10.0 does not contain a valid hub URL"),
		),
		Entry("missing image tag",
			fields{
				Pilot:      "docker.io/istio/pilot1.10.0",
				InstallCNI: "docker.io/istio/cni:1.10.0",
				ProxyV2:    "docker.io/istio/proxyv2:1.10.0",
				Ztunnel:    "docker.io/istio/ztunnel:1.10.0",
			},
			images.HubTag{},
			true,
			fmt.Errorf("image docker.io/istio/pilot1.10.0 does not contain a valid tag"),
		),
		Entry("images from different hubs",
			fields{
				Pilot:      "docker.io/istio/pilot:1.10.0",
				InstallCNI: "docker.io/istio/cni:1.10.0",
				ProxyV2:    "foo.bar/istio/proxyv2:1.10.0",
				Ztunnel:    "docker.io/istio/ztunnel:1.10.0",
			},
			images.HubTag{},
			true,
			fmt.Errorf("image foo.bar/istio/proxyv2:1.10.0 is not from the same hub as docker.io/istio/pilot:1.10.0"),
		),
		Entry("images with different tags",
			fields{
				Pilot:      "docker.io/istio/pilot:1.10.0",
				InstallCNI: "docker.io/istio/cni:1.10.0",
				ProxyV2:    "docker.io/istio/proxyv2:1.11.0",
				Ztunnel:    "docker.io/istio/ztunnel:1.10.0",
			},
			images.HubTag{},
			true,
			fmt.Errorf("image docker.io/istio/proxyv2:1.11.0 does not have the same tag as docker.io/istio/pilot:1.10.0"),
		),
		Entry("empty image",
			fields{
				Pilot:      "",
				InstallCNI: "docker.io/istio/cni:1.10.0",
				ProxyV2:    "docker.io/istio/proxyv2:1.10.0",
			},
			images.HubTag{},
			true,
			fmt.Errorf("image can not be empty"),
		),
	)
})

var _ = Describe("Images.GetFipsImages", func() {
	_ = os.Setenv("pilot", "docker.io/istio/pilot:1.10.0")
	_ = os.Setenv("install-cni", "docker.io/istio/cni:1.10.0")
	_ = os.Setenv("proxyv2", "docker.io/istio/proxyv2:1.10.0")
	_ = os.Setenv("ztunnel", "docker.io/istio/ztunnel:1.10.0")

	Context("when KYMA_FIPS_MODE_ENABLED is true", func() {
		It("should set the FIPS images", func() {
			_ = os.Setenv("KYMA_FIPS_MODE_ENABLED", "true")
			_ = os.Setenv("pilot-fips", "docker.io/istio/pilot-fips:1.10.0")
			_ = os.Setenv("install-cni-fips", "docker.io/istio/cni-fips:1.10.0")
			_ = os.Setenv("proxyv2-fips", "docker.io/istio/proxyv2-fips:1.10.0")
			_ = os.Setenv("ztunnel-fips", "docker.io/istio/ztunnel-fips:1.10.0")

			e, err := images.GetImages()
			Expect(err).NotTo(HaveOccurred())
			Expect(e.Pilot).To(Equal(images.Image("docker.io/istio/pilot-fips:1.10.0")))
			Expect(e.InstallCNI).To(Equal(images.Image("docker.io/istio/cni-fips:1.10.0")))
			Expect(e.ProxyV2).To(Equal(images.Image("docker.io/istio/proxyv2-fips:1.10.0")))
			Expect(e.Ztunnel).To(Equal(images.Image("docker.io/istio/ztunnel-fips:1.10.0")))
		})

		It("should return an error when FIPS image environment variables are missing", func() {
			_ = os.Setenv("KYMA_FIPS_MODE_ENABLED", "true")
			_ = os.Unsetenv("pilot-fips")
			_ = os.Unsetenv("install-cni-fips")
			_ = os.Unsetenv("proxyv2-fips")
			_ = os.Unsetenv("ztunnel-fips")

			_, err := images.GetImages()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("environment variable \"pilot-fips\" should not be empty"))
			Expect(err.Error()).To(ContainSubstring("environment variable \"install-cni-fips\" should not be empty"))
			Expect(err.Error()).To(ContainSubstring("environment variable \"proxyv2-fips\" should not be empty"))
			Expect(err.Error()).To(ContainSubstring("environment variable \"ztunnel-fips\" should not be empty"))
		})
	})

	Context("when KYMA_FIPS_MODE_ENABLED is false", func() {
		It("should use standard images", func() {
			_ = os.Setenv("KYMA_FIPS_MODE_ENABLED", "false")
			_ = os.Setenv("pilot-fips", "docker.io/istio/pilot-fips:1.10.0")
			_ = os.Setenv("install-cni-fips", "docker.io/istio/cni-fips:1.10.0")
			_ = os.Setenv("proxyv2-fips", "docker.io/istio/proxyv2-fips:1.10.0")
			_ = os.Setenv("ztunnel-fips", "docker.io/istio/ztunnel-fips:1.10.0")

			e, err := images.GetImages()
			Expect(err).NotTo(HaveOccurred())
			Expect(e.Pilot).To(Equal(images.Image("docker.io/istio/pilot:1.10.0")))
			Expect(e.InstallCNI).To(Equal(images.Image("docker.io/istio/cni:1.10.0")))
			Expect(e.ProxyV2).To(Equal(images.Image("docker.io/istio/proxyv2:1.10.0")))
			Expect(e.Ztunnel).To(Equal(images.Image("docker.io/istio/ztunnel:1.10.0")))

		})
	})
})
