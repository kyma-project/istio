package istio_resources_test

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/istio_resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"os"
	"sigs.k8s.io/yaml"
)

var _ = Describe("ApplyResources", func() {
	Context("k3d", func() {
		It("should create resource and return true for changed", func() {

			client := createFakeClient()

			sample := istio_resources.NewSampleResource()

			//when
			changed, err := sample.Apply(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(BeTrue())
		})

		It("should return false for changed if no change is needed", func() {
			//given
			manifest, err := os.ReadFile("sample.yaml")
			Expect(err).To(Not(HaveOccurred()))

			var filter networkingv1alpha3.EnvoyFilter
			err = yaml.Unmarshal(manifest, &filter)
			Expect(err).To(Not(HaveOccurred()))

			client := createFakeClient(&filter)

			sample := istio_resources.NewSampleResource()

			//when
			changed, err := sample.Apply(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(BeFalse())
		})

		It("should return true for changed if change is needed", func() {
			//given
			manifest, err := os.ReadFile("sample.yaml")
			Expect(err).To(Not(HaveOccurred()))

			var filter networkingv1alpha3.EnvoyFilter
			err = yaml.Unmarshal(manifest, &filter)
			Expect(err).To(Not(HaveOccurred()))

			filter.Spec.Priority = 2
			client := createFakeClient(&filter)

			sample := istio_resources.NewSampleResource()

			//when
			changed, err := sample.Apply(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(BeTrue())
		})
	})
})
