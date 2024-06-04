package main

import (
  "testing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestManifest(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Manifest Suite")
}

var _ = Describe("External Install", func() {
	Context("Build Options", func() {
		It("should build correct option flag for compatibilityVersion", func() {
      istioVersion := "1.21.2"
      res, err := buildCompatibilityOption([]string{}, istioVersion)
      Expect(err).ToNot(HaveOccurred())
      Expect(res[0]).To(Equal("compatibilityVersion=1.20"))
    })

		It("should fail if version without semver is passed", func() {
      var nilSlice []string
      istioVersion := "1.21"
      istioVersion2 := "1"
      istioVersion3 := "1.21.2.4"

      res, err := buildCompatibilityOption([]string{}, istioVersion)
      Expect(err).To(HaveOccurred())
      Expect(res).To(Equal(nilSlice))

      res, err = buildCompatibilityOption([]string{}, istioVersion2)
      Expect(err).To(HaveOccurred())
      Expect(res).To(Equal(nilSlice))

      res, err = buildCompatibilityOption([]string{}, istioVersion3)
      Expect(err).To(HaveOccurred())
      Expect(res).To(Equal(nilSlice))
    })

		It("should fail if random string is passed", func() {
      var nilSlice []string
      istioVersion := "randomString"

      res, err := buildCompatibilityOption([]string{}, istioVersion)
      Expect(err).To(HaveOccurred())
      Expect(res).To(Equal(nilSlice))
    })
  })
})
