package istio_resources

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createFakeClient(objects ...client.Object) client.Client {
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
