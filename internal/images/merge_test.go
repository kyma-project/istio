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
			func(input string, registryAndTag images.RegistryAndTag, expectedHub string, expectedTag string, expectsError bool) {
				out, err := images.MergeRegistryAndTagConfiguration([]byte(input), registryAndTag)

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
				images.RegistryAndTag{Registry: "my-hub", Tag: "my-tag"},
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
				images.RegistryAndTag{Registry: "new-hub", Tag: "new-tag"},
				"new-hub",
				"new-tag",
				false,
			),

			Entry("fails on invalid yaml",
				`::: bad yaml :::`,
				images.RegistryAndTag{},
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
})
