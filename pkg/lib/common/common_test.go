package common_test

import (
	"time"

	"github.com/kyma-project/istio/operator/pkg/lib/common"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AddRestartAnnotation", func() {
	It("should add restart annotation when initially there are none", func() {
		// when
		annotations := common.AddRestartAnnotation(nil)

		// then
		Expect(annotations).To(HaveLen(1))
		Expect(annotations).To(HaveKey("istio-operator.kyma-project.io/restartedAt"))
	})

	It("should add restart annotation to an existing annotations", func() {
		// given
		annotations := map[string]string{"some_annotation": "blah"}

		// when
		common.AddRestartAnnotation(annotations)

		// then
		Expect(annotations).To(HaveLen(2))
		Expect(annotations).To(HaveKey("istio-operator.kyma-project.io/restartedAt"))
	})
})

var _ = Describe("WasRestarted", func() {
	It("should return false when restart annotation is missing", func() {
		// when
		wasRestarted := common.WasRestarted(nil)

		// then
		Expect(wasRestarted).To(BeTrue())
	})

	It("should return true when restart annotation is in map", func() {
		// given
		annotations := map[string]string{"istio-operator.kyma-project.io/restartedAt": time.RFC3339}

		// when
		wasRestarted := common.WasRestarted(annotations)

		// then
		Expect(wasRestarted).To(BeTrue())
	})
})
