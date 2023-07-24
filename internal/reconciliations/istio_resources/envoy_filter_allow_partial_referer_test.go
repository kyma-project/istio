package istio_resources

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Apply", func() {
	Context("k3d", func() {
		It("should return created if no resource was present", func() {
			client := createFakeClient()
			sample := NewEnvoyFilterAllowPartialReferer()

			//when
			changed, err := sample.apply(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultCreated))
		})

		It("should return not changed if no change is needed", func() {
			//given
			var filter networkingv1alpha3.EnvoyFilter
			err := yaml.Unmarshal(manifest, &filter)
			Expect(err).To(Not(HaveOccurred()))

			client := createFakeClient(&filter)

			sample := NewEnvoyFilterAllowPartialReferer()

			//when
			changed, err := sample.apply(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultNone))
		})

		It("should return updated if change is needed", func() {
			//given
			var filter networkingv1alpha3.EnvoyFilter
			err := yaml.Unmarshal(manifest, &filter)
			Expect(err).To(Not(HaveOccurred()))

			filter.Spec.Priority = 2
			client := createFakeClient(&filter)

			sample := NewEnvoyFilterAllowPartialReferer()

			//when
			changed, err := sample.apply(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultUpdated))
		})
	})
})
