package resources_test

import (
	"context"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/resources"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Disclaimer annotation", func() {
	It("Should annotate with disclaimer when there was no such annotation", func() {

		unstr := unstructured.Unstructured{Object: map[string]interface{}{}}
		unstr.SetName("test")
		unstr.SetKind("ConfigMap")
		unstr.SetAPIVersion("v1")

		client := createFakeClient(&unstr)

		Expect(resources.AnnotateWithDisclaimer(context.Background(), &unstr, client)).Should(Succeed())

		Expect(client.Get(context.Background(), ctrlClient.ObjectKey{Name: unstr.GetName()}, &unstr)).To(Succeed())
		anns := unstr.GetAnnotations()
		Expect(anns[resources.DisclaimerKey]).To(Equal(resources.DisclaimerValue))
	})

	It("should return true if there is managed by disclaimer annotation", func() {

		unstr := unstructured.Unstructured{Object: map[string]interface{}{}}
		unstr.SetName("test")
		unstr.SetKind("ConfigMap")
		unstr.SetAPIVersion("v1")

		client := createFakeClient(&unstr)

		Expect(resources.AnnotateWithDisclaimer(context.Background(), &unstr, client)).Should(Succeed())
		Expect(client.Get(context.Background(), ctrlClient.ObjectKey{Name: unstr.GetName()}, &unstr)).To(Succeed())
		Expect(resources.HasManagedByDisclaimer(unstr)).To(BeTrue())
	})

	It("should return false if there is no managed by disclaimer annotation", func() {

		unstr := unstructured.Unstructured{Object: map[string]interface{}{}}
		unstr.SetName("test")
		unstr.SetKind("ConfigMap")
		unstr.SetAPIVersion("v1")

		client := createFakeClient(&unstr)

		Expect(client.Get(context.Background(), ctrlClient.ObjectKey{Name: unstr.GetName()}, &unstr)).To(Succeed())
		Expect(resources.HasManagedByDisclaimer(unstr)).To(BeFalse())
	})
})

func createFakeClient(objects ...ctrlClient.Object) ctrlClient.Client {
	err := operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1alpha3.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = securityv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1beta1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
