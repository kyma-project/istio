package images_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"sigs.k8s.io/yaml"

	"github.com/kyma-project/istio/operator/internal/images"
)

var _ = Describe("Images merging", func() {

	Describe("MergeHubConfiguration", func() {

		DescribeTable("merges hub correctly",
			func(input string, hub string, expectedHub string, expectsError bool) {
				out, err := images.MergeHubConfiguration([]byte(input), hub)

				if expectsError {
					Expect(err).To(HaveOccurred())
					return
				}

				Expect(err).NotTo(HaveOccurred())

				var parsed map[string]interface{}
				Expect(yaml.Unmarshal(out, &parsed)).To(Succeed())

				spec := parsed["spec"].(map[string]interface{})
				Expect(spec["hub"]).To(Equal(expectedHub))
			},

			Entry("adds hub when missing",
				`
spec:
  profile: default
`,
				"my-hub",
				"my-hub",
				false,
			),

			Entry("overrides existing hub",
				`
spec:
  hub: old-hub
`,
				"new-hub",
				"new-hub",
				false,
			),

			Entry("fails on invalid yaml",
				`::: bad yaml :::`,
				"hub",
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
