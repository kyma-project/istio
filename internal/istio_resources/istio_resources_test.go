package istio_resources_test

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/istio_resources"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ApplyResources", func() {
	Context("k3d", func() {
		It("should create resource and return true for both restarts", func() {

			client := createFakeClient()

			sample := istio_resources.NewSampleResource()

			isResources := istio_resources.IstioResources{}
			isResources.AddResourceToList(sample)

			//when
			igRestart, proxyRestart, err := isResources.ApplyResources(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(igRestart).To(BeTrue())
			Expect(proxyRestart).To(BeTrue())
		})
	})
})

func createFakeClient(objects ...client.Object) client.Client {
	err := operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1alpha3.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
