//go:build experimental

package istioresources

import (
	"context"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	securityv1 "istio.io/client-go/pkg/apis/security/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Reconciliation", func() {
	numTrustedProxies := 1
	istioCR := operatorv1alpha2.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
			UID:             "1234",
		},
		Spec: operatorv1alpha2.IstioSpec{
			Config: operatorv1alpha2.Config{
				NumTrustedProxies: &numTrustedProxies,
			},
		},
	}

	It("should be created when hyperscaler is AWS, and the dual stack is enabled", func() {
		//given
		n := corev1.Node{Spec: corev1.NodeSpec{ProviderID: "aws://asdasdads"}}

		kymaProvisioningInfoCM := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyma-provisioning-info",
				Namespace: "kyma-system",
			},
			Data: map[string]string{
				"details": `
  networkDetails:
   dualStackIPEnabled: true
`,
			},
		}

		client := createFakeClient(&n, &kymaProvisioningInfoCM)
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.Background(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(client.Get(context.Background(), ctrlclient.ObjectKey{Name: "proxy-protocol", Namespace: "istio-system"}, &networkingv1alpha3.EnvoyFilter{})).Should(Succeed())
	})

	It("should not be created when hyperscaler is AWS, even if proxy-protocol is set but the dual stack is not enabled", func() {
		//given
		n := corev1.Node{Spec: corev1.NodeSpec{ProviderID: "aws://asdasdads"}}
		kymaProvisioningInfoCM := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyma-provisioning-info",
				Namespace: "kyma-system",
			},
			Data: map[string]string{
				"details": `
  networkDetails:
   dualStackIPEnabled: false
`,
			},
		}

		client := createFakeClient(&n, &kymaProvisioningInfoCM)
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.Background(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))

		var e networkingv1alpha3.EnvoyFilter
		Expect(client.Get(context.Background(), ctrlclient.ObjectKey{Name: "proxy-protocol", Namespace: "istio-system"}, &e)).Should(Not(Succeed()))
	})

})

func createFakeClient(objects ...ctrlclient.Object) ctrlclient.Client {
	err := operatorv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1alpha3.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = securityv1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}
