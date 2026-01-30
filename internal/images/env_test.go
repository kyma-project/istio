package images_test

import (
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

var _ = Describe("GetImages", func() {
	BeforeEach(func() {
		_ = os.Unsetenv("KYMA_FIPS_MODE_ENABLED")
	})

	DescribeTable("parses images from environment variables",
		func(envVars map[string]string, expectedRegistry string, expectedTag string, wantErr bool, errSubstring string) {
			// Set up environment variables
			for key, value := range envVars {
				if value != "" {
					_ = os.Setenv(key, value)
				} else {
					_ = os.Unsetenv(key)
				}
			}

			// Clean up after test
			defer func() {
				for key := range envVars {
					_ = os.Unsetenv(key)
				}
			}()

			e, err := images.GetImages()
			if wantErr {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring(errSubstring))
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(e.Registry).To(Equal(expectedRegistry))
				Expect(e.Tag).To(Equal(expectedTag))
			}
		},
		Entry("valid images",
			map[string]string{
				"pilot":       "docker.io/istio/pilot:1.10.0",
				"install-cni": "docker.io/istio/cni:1.10.0",
				"proxyv2":     "docker.io/istio/proxyv2:1.10.0",
				"ztunnel":     "docker.io/istio/ztunnel:1.10.0",
			},
			"docker.io/istio",
			"1.10.0",
			false,
			"",
		),
		Entry("invalid image hub",
			map[string]string{
				"pilot":       "pilot:1.10.0",
				"install-cni": "docker.io/istio/cni:1.10.0",
				"proxyv2":     "docker.io/istio/proxyv2:1.10.0",
				"ztunnel":     "docker.io/istio/ztunnel:1.10.0",
			},
			"",
			"",
			true,
			"does not contain a valid format",
		),
		Entry("missing image tag",
			map[string]string{
				"pilot":       "docker.io/istio/pilot1.10.0",
				"install-cni": "docker.io/istio/cni:1.10.0",
				"proxyv2":     "docker.io/istio/proxyv2:1.10.0",
				"ztunnel":     "docker.io/istio/ztunnel:1.10.0",
			},
			"",
			"",
			true,
			"does not contain a valid tag",
		),
		Entry("images from different hubs",
			map[string]string{
				"pilot":       "docker.io/istio/pilot:1.10.0",
				"install-cni": "docker.io/istio/cni:1.10.0",
				"proxyv2":     "foo.bar/istio/proxyv2:1.10.0",
				"ztunnel":     "docker.io/istio/ztunnel:1.10.0",
			},
			"",
			"",
			true,
			"is not from the same hub",
		),
		Entry("images with different tags",
			map[string]string{
				"pilot":       "docker.io/istio/pilot:1.10.0",
				"install-cni": "docker.io/istio/cni:1.10.0",
				"proxyv2":     "docker.io/istio/proxyv2:1.11.0",
				"ztunnel":     "docker.io/istio/ztunnel:1.10.0",
			},
			"",
			"",
			true,
			"does not have the same tag",
		),
		Entry("empty pilot image",
			map[string]string{
				"pilot":       "",
				"install-cni": "docker.io/istio/cni:1.10.0",
				"proxyv2":     "docker.io/istio/proxyv2:1.10.0",
				"ztunnel":     "docker.io/istio/ztunnel:1.10.0",
			},
			"",
			"",
			true,
			"pilot",
		),
		Entry("ztunnel is empty",
			map[string]string{
				"pilot":       "docker.io/istio/pilot:1.10.0",
				"install-cni": "docker.io/istio/cni:1.10.0",
				"proxyv2":     "docker.io/istio/proxyv2:1.10.0",
				"ztunnel":     "",
			},
			"docker.io/istio",
			"1.10.0",
			false,
			"",
		),
		Entry("images with digest suffix",
			map[string]string{
				"pilot":       "docker.io/istio/pilot:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703822e04d33910",
				"install-cni": "docker.io/istio/cni:1.10.0@sha256:90638cf608f9c5dc4b67062a4ssdassa23a21199d6b6214b7703822e04d33910",
				"proxyv2":     "docker.io/istio/proxyv2:1.10.0@sha256:90638casdsdsdb67062a44dc60fa23a21199d6b6214b7703822e04d33910",
				"ztunnel":     "docker.io/istio/ztunnel:1.10.0@sha256:90638cf608f9c5dc4b67062a44dcasdasdad3a21199d6b6214b7703822e04d33910",
			},
			"docker.io/istio",
			"1.10.0",
			false,
			"",
		),
		Entry("registry with port",
			map[string]string{
				"pilot":       "localhost:5000/istio/pilot:1.10.0",
				"install-cni": "localhost:5000/istio/cni:1.10.0",
				"proxyv2":     "localhost:5000/istio/proxyv2:1.10.0",
				"ztunnel":     "localhost:5000/istio/ztunnel:1.10.0",
			},
			"localhost:5000/istio",
			"1.10.0",
			false,
			"",
		),
		Entry("registry with port images with digest suffix",
			map[string]string{
				"pilot":       "docker.io:9000/istio/pilot:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703822e04d33910",
				"install-cni": "docker.io:9000/istio/cni:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703822e04d33910",
				"proxyv2":     "docker.io:9000/istio/proxyv2:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703822e04d33910",
				"ztunnel":     "docker.io:9000/istio/ztunnel:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703822e04d33910",
			},
			"docker.io:9000/istio",
			"1.10.0",
			false,
			"",
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
			Expect(e.Pilot).To(Equal(images.Image{Registry: "docker.io/istio", Name: "pilot-fips", Tag: "1.10.0"}))
			Expect(e.InstallCNI).To(Equal(images.Image{Registry: "docker.io/istio", Name: "cni-fips", Tag: "1.10.0"}))
			Expect(e.ProxyV2).To(Equal(images.Image{Registry: "docker.io/istio", Name: "proxyv2-fips", Tag: "1.10.0"}))
			Expect(e.Ztunnel).To(Equal(images.Image{Registry: "docker.io/istio", Name: "ztunnel-fips", Tag: "1.10.0"}))
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
			Expect(err.Error()).NotTo(ContainSubstring("environment variable \"ztunnel-fips\" should not be empty"))
		})
	})

	Context("when KYMA_FIPS_MODE_ENABLED is false", func() {
		It("should use standard images", func() {
			_ = os.Setenv("KYMA_FIPS_MODE_ENABLED", "false")

			_ = os.Setenv("pilot-fips", "docker.io/istio/pilot-fips:1.10.0")
			_ = os.Setenv("install-cni-fips", "docker.io/istio/cni-fips:1.10.0")
			_ = os.Setenv("proxyv2-fips", "docker.io/istio/proxyv2-fips:1.10.0")
			_ = os.Setenv("ztunnel-fips", "docker.io/istio/ztunnel-fips:1.10.0")

			_ = os.Setenv("pilot", "docker.io/istio/pilot:1.10.0")
			_ = os.Setenv("install-cni", "docker.io/istio/cni:1.10.0")
			_ = os.Setenv("proxyv2", "docker.io/istio/proxyv2:1.10.0")
			_ = os.Setenv("ztunnel", "docker.io/istio/ztunnel:1.10.0")

			e, err := images.GetImages()
			Expect(err).NotTo(HaveOccurred())
			Expect(e.Pilot).To(Equal(images.Image{Registry: "docker.io/istio", Name: "pilot", Tag: "1.10.0"}))
			Expect(e.InstallCNI).To(Equal(images.Image{Registry: "docker.io/istio", Name: "cni", Tag: "1.10.0"}))
			Expect(e.ProxyV2).To(Equal(images.Image{Registry: "docker.io/istio", Name: "proxyv2", Tag: "1.10.0"}))
			Expect(e.Ztunnel).To(Equal(images.Image{Registry: "docker.io/istio", Name: "ztunnel", Tag: "1.10.0"}))

		})
	})
})

var _ = Describe("Images Test", func() {
	It("should return error when required envs are missing", func() {
		_ = os.Unsetenv("pilot")
		_ = os.Unsetenv("install-cni")
		_ = os.Unsetenv("proxyv2")

		_, err := images.GetImages()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("environment variable \"pilot\" should not be empty"))
		Expect(err.Error()).To(ContainSubstring("environment variable \"install-cni\" should not be empty"))
		Expect(err.Error()).To(ContainSubstring("environment variable \"proxyv2\" should not be empty"))
	})
	It("should return images when all required envs are set", func() {
		_ = os.Setenv("pilot", "docker.io/istio/pilot:1.10.0")
		_ = os.Setenv("install-cni", "docker.io/istio/cni:1.10.0")
		_ = os.Setenv("proxyv2", "docker.io/istio/proxyv2:1.10.0")

		e, err := images.GetImages()
		Expect(err).NotTo(HaveOccurred())
		Expect(e.Pilot).To(Equal(images.Image{Registry: "docker.io/istio", Name: "pilot", Tag: "1.10.0"}))
		Expect(e.InstallCNI).To(Equal(images.Image{Registry: "docker.io/istio", Name: "cni", Tag: "1.10.0"}))
		Expect(e.ProxyV2).To(Equal(images.Image{Registry: "docker.io/istio", Name: "proxyv2", Tag: "1.10.0"}))
	})
	It("should return valid registry", func() {
		_ = os.Setenv("pilot", "docker.io/istio/pilot:1.10.0")
		_ = os.Setenv("install-cni", "docker.io/istio/cni:1.10.0")
		_ = os.Setenv("proxyv2", "docker.io/istio/proxyv2:1.10.0")

		e, err := images.GetImages()
		Expect(err).NotTo(HaveOccurred())

		registry := e.Pilot.GetHub()
		Expect(registry).To(Equal("docker.io/istio"))
	})
	It("should return valid registry with port", func() {
		_ = os.Setenv("pilot", "docker.io:9000/istio/pilot:1.10.0")
		_ = os.Setenv("install-cni", "docker.io:9000/istio/cni:1.10.0")
		_ = os.Setenv("proxyv2", "docker.io:9000/istio/proxyv2:1.10.0")
		_ = os.Unsetenv("ztunnel")

		e, err := images.GetImages()
		Expect(err).NotTo(HaveOccurred())

		registry := e.Pilot.GetHub()
		Expect(registry).To(Equal("docker.io:9000/istio"))
	})
	It("should return valid tag", func() {
		_ = os.Setenv("pilot", "docker.io/istio/pilot:1.10.0")
		_ = os.Setenv("install-cni", "docker.io/istio/cni:1.10.0")
		_ = os.Setenv("proxyv2", "docker.io/istio/proxyv2:1.10.0")

		e, err := images.GetImages()
		Expect(err).NotTo(HaveOccurred())

		tag := e.Pilot.GetTag()
		Expect(tag).To(Equal("1.10.0"))
	})
	It("should return valid tag with sha", func() {
		_ = os.Setenv("pilot", "docker.io/istio/pilot:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703822e04d33910")
		_ = os.Setenv("install-cni", "docker.io/istio/cni:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703aaaad33910")
		_ = os.Setenv("proxyv2", "docker.io/istio/proxyv2:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703fffff04d33910")

		e, err := images.GetImages()
		Expect(err).NotTo(HaveOccurred())

		tag := e.Pilot.GetTag()
		Expect(tag).To(Equal("1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703822e04d33910"))
	})
	It("should return valid image name", func() {
		_ = os.Setenv("pilot", "docker.io/istio/pilot:1.10.0")
		_ = os.Setenv("install-cni", "docker.io/istio/cni:1.10.0")
		_ = os.Setenv("proxyv2", "docker.io/istio/proxyv2:1.10.0")

		e, err := images.GetImages()
		Expect(err).NotTo(HaveOccurred())

		name := e.Pilot.GetName()
		Expect(name).To(Equal("pilot"))
	})
	It("should return valid image name with registry port", func() {
		_ = os.Setenv("pilot", "docker.io:9000/istio/pilot:1.10.0")
		_ = os.Setenv("install-cni", "docker.io:9000/istio/cni:1.10.0")
		_ = os.Setenv("proxyv2", "docker.io:9000/istio/proxyv2:1.10.0")
		_ = os.Unsetenv("ztunnel")

		e, err := images.GetImages()
		Expect(err).NotTo(HaveOccurred())

		name := e.Pilot.GetName()
		Expect(name).To(Equal("pilot"))
	})
	It("should return valid image name with digest", func() {
		_ = os.Setenv("pilot", "docker.io/istio/pilot:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703822e04d33910")
		_ = os.Setenv("install-cni", "docker.io/istio/cni:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703aaaad33910")
		_ = os.Setenv("proxyv2", "docker.io/istio/proxyv2:1.10.0@sha256:90638cf608f9c5dc4b67062a44dc60fa23a21199d6b6214b7703fffff04d33910")

		e, err := images.GetImages()
		Expect(err).NotTo(HaveOccurred())

		name := e.Pilot.GetName()
		Expect(name).To(Equal("pilot"))
	})

})
