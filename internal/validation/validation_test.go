package validation_test

import (
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/kyma-project/istio/operator/internal/validation"
	"github.com/onsi/ginkgo/v2/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestValidation(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Validation suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("validation-suite", report)
})

var _ = Describe("Validation", func() {
	It("should successfully validate authorizers if no issue is present", func() {
		//given
		istioCr := istioCR.Istio{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: istioCR.IstioSpec{
				Config: istioCR.Config{
					Authorizers: []*istioCR.Authorizer{
						{
							Name:    "test-authorizer",
							Service: "test",
							Port:    2318,
						},
						{
							Name:    "test-authorizer1",
							Service: "test2",
							Port:    2317,
						},
					},
				},
			},
		}
		//when
		err := validation.ValidateAuthorizers(istioCr)

		//then
		Expect(err).NotTo(HaveOccurred())
	})

	It("should fail to validate if some authorizers have the same name", func() {
		//given
		istioCr := istioCR.Istio{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: istioCR.IstioSpec{
				Config: istioCR.Config{
					Authorizers: []*istioCR.Authorizer{
						{
							Name:    "test-authorizer",
							Service: "test",
							Port:    2318,
						},
						{
							Name:    "test-authorizer",
							Service: "test2",
							Port:    2317,
						},
					},
				},
			},
		}
		//when
		err := validation.ValidateAuthorizers(istioCr)

		//then
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(Equal("test-authorizer is duplicated"))
	})

})
