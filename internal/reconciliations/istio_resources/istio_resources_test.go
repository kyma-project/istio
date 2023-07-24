package istio_resources

import (
	"context"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Reconcilation", func() {
	Context("k3d", func() {
		It("should create resource and return no error", func() {

			client := createFakeClient()

			sample := NewEnvoyFilterAllowPartialReferer()

			reconciler := NewReconciler(client, []Resource{sample})

			//when
			err := reconciler.Reconcile(context.TODO())

			//then
			Expect(err).To(Not(HaveOccurred()))

			var s networkingv1alpha3.EnvoyFilterList
			listErr := client.List(context.TODO(), &s)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(s.Items).To(HaveLen(1))
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
