package images_test

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/istio/operator/internal/images"
)

func TestEnvs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Environment Suite")
}

var _ = Describe("Images.GetHub", func() {
	type fields struct {
		Pilot      images.Image
		InstallCNI images.Image
		ProxyV2    images.Image
		Ztunnel    images.Image
	}

	DescribeTable("GetHub",
		func(f fields, want string, wantErr bool, err error) {
			e := &images.Images{
				Pilot:      f.Pilot,
				InstallCNI: f.InstallCNI,
				ProxyV2:    f.ProxyV2,
				Ztunnel:    f.Ztunnel,
			}
			got, err := e.GetHub()
			if wantErr {
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("image"))
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
			"docker.io/istio",
			false,
			nil,
		),
		Entry("invalid image format",
			fields{
				Pilot:      "pilot:1.10.0",
				InstallCNI: "docker.io/istio/cni:1.10.0",
				ProxyV2:    "docker.io/istio/proxyv2:1.10.0",
				Ztunnel:    "docker.io/istio/ztunnel:1.10.0",
			},
			"",
			true,
			fmt.Errorf("image pilot:1.10.0 does not contain a valid hub URL"),
		),
		Entry("images from different hubs",
			fields{
				Pilot:      "docker.io/istio/pilot:1.10.0",
				InstallCNI: "docker.io/istio/cni:1.10.0",
				ProxyV2:    "foo.bar/istio/proxyv2:1.10.0",
				Ztunnel:    "docker.io/istio/ztunnel:1.10.0",
			},
			"",
			true,
			fmt.Errorf("image foo.bar/istio/proxyv2:1.10.0 is not from the same hub as docker.io/istio/pilot:1.10.0"),
		),
	)
})
