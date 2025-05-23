package annotations_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"

	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Annotations Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("annotations-suite", report)
})

var _ = Describe("AddRestartAnnotation", func() {
	It("should add restart annotation when initially there are none", func() {
		// when
		anns := annotations.AddRestartAnnotation(nil)

		// then
		Expect(anns).To(HaveLen(1))
		Expect(anns).To(HaveKey("istio-operator.kyma-project.io/restartedAt"))
	})

	It("should add restart annotation to an existing annotations", func() {
		// given
		anns := map[string]string{"some_annotation": "blah"}

		// when
		annotations.AddRestartAnnotation(anns)

		// then
		Expect(anns).To(HaveLen(2))
		Expect(anns).To(HaveKey("istio-operator.kyma-project.io/restartedAt"))
	})
})

var _ = Describe("HasRestartAnnotation", func() {
	It("should return false when restart annotation is missing", func() {
		// given
		anns := map[string]string{}

		// when
		hasRestartAnnotation := annotations.HasRestartAnnotation(anns)

		// then
		Expect(hasRestartAnnotation).To(BeFalse())
	})

	It("should return true when restart annotation is in map", func() {
		// given
		anns := map[string]string{"istio-operator.kyma-project.io/restartedAt": time.RFC3339}

		// when
		hasRestartAnnotation := annotations.HasRestartAnnotation(anns)

		// then
		Expect(hasRestartAnnotation).To(BeTrue())
	})
})
