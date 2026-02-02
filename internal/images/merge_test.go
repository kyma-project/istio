package images_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/yaml"

	"github.com/kyma-project/istio/operator/internal/images"
)

var _ = Describe("Images merging", func() {

	Describe("MergeRegistryAndTagConfiguration", func() {

		DescribeTable("merges hub correctly",
			func(input string, img images.Images, expectedHub string, expectedTag string, expectsError bool) {
				out, err := images.MergeComponentImages([]byte(input), img)

				if expectsError {
					Expect(err).To(HaveOccurred())
					return
				}

				Expect(err).NotTo(HaveOccurred())

				var parsed map[string]interface{}
				Expect(yaml.Unmarshal(out, &parsed)).To(Succeed())

				spec := parsed["spec"].(map[string]interface{})
				Expect(spec["hub"]).To(Equal(expectedHub))
				Expect(spec["tag"]).To(Equal(expectedTag))
			},

			Entry("adds hub when missing",
				`
spec:
  profile: default
`,
				images.Images{
					Registry:   "my-hub",
					Tag:        "my-tag",
					Pilot:      images.Image{Registry: "my-hub", Name: "my-pilot", Tag: "my-tag"},
					InstallCNI: images.Image{Registry: "my-hub", Name: "my-cni", Tag: "my-tag"},
					ProxyV2:    images.Image{Registry: "my-hub", Name: "my-proxy", Tag: "my-tag"},
					Ztunnel:    images.Image{Registry: "my-hub", Name: "my-ztunnel", Tag: "my-tag"},
				},
				"my-hub",
				"my-tag",
				false,
			),

			Entry("overrides existing hub",
				`
spec:
  hub: old-hub
  tag: old-tag
`,
				images.Images{
					Registry:   "new-hub",
					Tag:        "new-tag",
					Pilot:      images.Image{Registry: "new-hub", Name: "my-pilot", Tag: "new-tag"},
					InstallCNI: images.Image{Registry: "new-hub", Name: "my-cni", Tag: "new-tag"},
					ProxyV2:    images.Image{Registry: "new-hub", Name: "my-proxy", Tag: "new-tag"},
					Ztunnel:    images.Image{Registry: "new-hub", Name: "my-ztunnel", Tag: "new-tag"},
				},
				"new-hub",
				"new-tag",
				false,
			),

			Entry("fails on invalid yaml",
				`::: bad yaml :::`,
				images.Images{},
				"",
				"",
				true,
			),
		)
	})

	Describe("MergePullSecretEnv", func() {

		BeforeEach(func() {
			_ = os.Unsetenv("SKR_IMG_PULL_SECRET")
		})

		AfterEach(func() {
			_ = os.Unsetenv("SKR_IMG_PULL_SECRET")
		})

		DescribeTable("handles pull secret correctly",
			func(input string, envValue string, expectedSecrets []interface{}) {
				if envValue != "" {
					Expect(os.Setenv("SKR_IMG_PULL_SECRET", envValue)).To(Succeed())
				}

				out, err := images.MergePullSecretEnv([]byte(input))
				Expect(err).NotTo(HaveOccurred())

				var parsed map[string]interface{}
				Expect(yaml.Unmarshal(out, &parsed)).To(Succeed())

				// No env var: manifest should remain unchanged
				if envValue == "" {
					Expect(string(out)).To(ContainSubstring("existing-secret"))
					return
				}

				spec := parsed["spec"].(map[string]interface{})
				values := spec["values"].(map[string]interface{})
				global := values["global"].(map[string]interface{})
				ips := global["imagePullSecrets"].([]interface{})

				Expect(ips).To(Equal(expectedSecrets))
			},

			Entry("does nothing if env var is not set",
				`
spec:
  values:
    global:
      imagePullSecrets:
        - existing-secret
`,
				"",
				nil,
			),

			Entry("adds secret if not present",
				`
spec:
  values:
    global:
      imagePullSecrets:
        - existing-secret
`,
				"my-secret",
				[]interface{}{"existing-secret", "my-secret"},
			),

			Entry("does not duplicate existing secret",
				`
spec:
  values:
    global:
      imagePullSecrets:
        - existing-secret
`,
				"existing-secret",
				[]interface{}{"existing-secret"},
			),

			Entry("creates entire structure if missing",
				`{}`,
				"my-secret",
				[]interface{}{"my-secret"},
			),
		)
	})

	Describe("Merge configurable istio images ", func() {

		DescribeTable("merges component images correctly",
			func(input string, img images.Images, expectedPilot string, expectedCNI string, expectedProxy string, expectsError bool) {
				out, err := images.MergeComponentImages([]byte(input), img)

				if expectsError {
					Expect(err).To(HaveOccurred())
					return
				}

				Expect(err).NotTo(HaveOccurred())

				var parsed map[string]interface{}
				Expect(yaml.Unmarshal(out, &parsed)).To(Succeed())

				spec := parsed["spec"].(map[string]interface{})
				values := spec["values"].(map[string]interface{})

				// Check pilot image
				pilot := values["pilot"].(map[string]interface{})
				Expect(pilot["image"]).To(Equal(expectedPilot))

				// Check CNI image
				cni := values["cni"].(map[string]interface{})
				Expect(cni["image"]).To(Equal(expectedCNI))

				// Check proxy image
				global := values["global"].(map[string]interface{})
				proxy := global["proxy"].(map[string]interface{})
				Expect(proxy["image"]).To(Equal(expectedProxy))

				proxy_init := global["proxy_init"].(map[string]interface{})
				Expect(proxy_init["image"]).To(Equal(expectedProxy))
			},

			Entry("sets all component images when values section is empty",
				`
spec:
  profile: default
`,
				images.Images{
					Registry:   "my-hub",
					Tag:        "my-tag",
					Pilot:      images.Image{Registry: "my-hub", Name: "my-pilot", Tag: "my-tag"},
					InstallCNI: images.Image{Registry: "my-hub", Name: "my-cni", Tag: "my-tag"},
					ProxyV2:    images.Image{Registry: "my-hub", Name: "my-proxy", Tag: "my-tag"},
					Ztunnel:    images.Image{Registry: "my-hub", Name: "my-ztunnel", Tag: "my-tag"},
				},
				"my-hub/my-pilot:my-tag",
				"my-hub/my-cni:my-tag",
				"my-hub/my-proxy:my-tag",
				false,
			),

			Entry("overrides existing component images",
				`
spec:
  values:
    pilot:
      image: old-pilot
    cni:
      image: old-cni
    global:
      proxy:
        image: old-proxy
`,
				images.Images{
					Registry:   "new-hub",
					Tag:        "new-tag",
					Pilot:      images.Image{Registry: "new-hub", Name: "new-pilot", Tag: "new-tag"},
					InstallCNI: images.Image{Registry: "new-hub", Name: "new-cni", Tag: "new-tag"},
					ProxyV2:    images.Image{Registry: "new-hub", Name: "new-proxy", Tag: "new-tag"},
					Ztunnel:    images.Image{Registry: "new-hub", Name: "new-ztunnel", Tag: "new-tag"},
				},
				"new-hub/new-pilot:new-tag",
				"new-hub/new-cni:new-tag",
				"new-hub/new-proxy:new-tag",
				false,
			),

			Entry("preserves other component configuration while updating images",
				`
spec:
  values:
    pilot:
      image: old-pilot
      resources:
        requests:
          cpu: 100m
    cni:
      image: old-cni
      enabled: true
    global:
      proxy:
        image: old-proxy
        resources:
          requests:
            memory: 128Mi
`,
				images.Images{
					Registry:   "updated-hub",
					Tag:        "v1.0",
					Pilot:      images.Image{Registry: "updated-hub", Name: "updated-pilot", Tag: "v1.0"},
					InstallCNI: images.Image{Registry: "updated-hub", Name: "updated-cni", Tag: "v1.0"},
					ProxyV2:    images.Image{Registry: "updated-hub", Name: "updated-proxy", Tag: "v1.0"},
					Ztunnel:    images.Image{Registry: "updated-hub", Name: "updated-ztunnel", Tag: "v1.0"},
				},
				"updated-hub/updated-pilot:v1.0",
				"updated-hub/updated-cni:v1.0",
				"updated-hub/updated-proxy:v1.0",
				false,
			),

			Entry("handles different registries in image URLs",
				`
spec:
  profile: default
`,
				images.Images{
					Registry:   "registry.example.com/istio",
					Tag:        "1.20.0",
					Pilot:      images.Image{Registry: "registry.example.com/istio", Name: "pilot", Tag: "1.20.0"},
					InstallCNI: images.Image{Registry: "registry.example.com/istio", Name: "install-cni", Tag: "1.20.0"},
					ProxyV2:    images.Image{Registry: "registry.example.com/istio", Name: "proxyv2", Tag: "1.20.0"},
					Ztunnel:    images.Image{Registry: "registry.example.com/istio", Name: "ztunnel", Tag: "1.20.0"},
				},
				"registry.example.com/istio/pilot:1.20.0",
				"registry.example.com/istio/install-cni:1.20.0",
				"registry.example.com/istio/proxyv2:1.20.0",
				false,
			),
		)

		It("should merge both hub/tag and component images in complete manifest", func() {
			input := `
spec:
  profile: production
  values:
    pilot:
      resources:
        requests:
          cpu: 500m
`

			img := images.Images{
				Registry:   "production.registry.io/istio",
				Tag:        "1.21.0",
				Pilot:      images.Image{Registry: "production.registry.io/istio", Name: "pilot", Tag: "1.21.0"},
				InstallCNI: images.Image{Registry: "production.registry.io/istio", Name: "install-cni", Tag: "1.21.0"},
				ProxyV2:    images.Image{Registry: "production.registry.io/istio", Name: "proxyv2", Tag: "1.21.0"},
				Ztunnel:    images.Image{Registry: "production.registry.io/istio", Name: "ztunnel", Tag: "1.21.0"},
			}

			out, err := images.MergeComponentImages([]byte(input), img)
			Expect(err).NotTo(HaveOccurred())

			var parsed map[string]interface{}
			Expect(yaml.Unmarshal(out, &parsed)).To(Succeed())

			spec := parsed["spec"].(map[string]interface{})

			// Verify hub and tag are set
			Expect(spec["hub"]).To(Equal("production.registry.io/istio"))
			Expect(spec["tag"]).To(Equal("1.21.0"))

			// Verify component images are set
			values := spec["values"].(map[string]interface{})
			pilot := values["pilot"].(map[string]interface{})
			Expect(pilot["image"]).To(Equal("production.registry.io/istio/pilot:1.21.0"))

			cni := values["cni"].(map[string]interface{})
			Expect(cni["image"]).To(Equal("production.registry.io/istio/install-cni:1.21.0"))

			global := values["global"].(map[string]interface{})
			proxy := global["proxy"].(map[string]interface{})
			Expect(proxy["image"]).To(Equal("production.registry.io/istio/proxyv2:1.21.0"))

			// Verify other settings are preserved
			pilotResources := pilot["resources"].(map[string]interface{})
			pilotRequests := pilotResources["requests"].(map[string]interface{})
			Expect(pilotRequests["cpu"]).To(Equal("500m"))
		})
	})

})
