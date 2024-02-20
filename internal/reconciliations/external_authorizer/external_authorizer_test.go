package external_authorizer_test

import (
	"context"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/external_authorizer"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func createFakeClient(objects ...ctrlclient.Object) ctrlclient.Client {
	err := operatorv1alpha2.AddToScheme(scheme.Scheme)
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

var _ = Describe("Reconciliation", func() {
	const (
		expectedName    = "expected-name"
		expectedPort    = 1234
		expectedService = "a.svc.cluster.local"
	)

	istioCR := operatorv1alpha2.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
			UID:             "1234",
		},
		Spec: operatorv1alpha2.IstioSpec{
			Config: operatorv1alpha2.Config{
				Authorizers: []*operatorv1alpha2.Authorizer{
					{
						Name:    expectedName,
						Service: expectedService,
						Port:    expectedPort,
					},
				},
			},
		},
	}

	Context("ServiceEntry creation", func() {

		It("should succeed creating ServiceEntry", func() {
			//given
			client := createFakeClient()
			reconciler := external_authorizer.NewReconciler(client)

			//when
			err := reconciler.Reconcile(context.TODO(), istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			var s networkingv1alpha3.ServiceEntryList
			listErr := client.List(context.TODO(), &s)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(s.Items).To(HaveLen(1))
		})

		It("should succeed configuring ServiceEntry", func() {
			//given
			client := createFakeClient()
			reconciler := external_authorizer.NewReconciler(client)

			//when
			err := reconciler.Reconcile(context.TODO(), istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			var s networkingv1alpha3.ServiceEntryList
			listErr := client.List(context.TODO(), &s)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(s.Items).To(HaveLen(1))

			Expect(s.Items[0].Name).To(Equal(expectedName))

			Expect(s.Items[0].Spec.Ports).To(HaveLen(1))
			Expect(s.Items[0].Spec.Ports[0].Protocol).To(Equal("http"))
			Expect(s.Items[0].Spec.Ports[0].Name).To(Equal("http"))
			Expect(s.Items[0].Spec.Ports[0].Number).To(BeEquivalentTo(expectedPort))

			Expect(s.Items[0].Spec.Hosts).To(HaveLen(1))
			Expect(s.Items[0].Spec.Hosts[0]).To(Equal(expectedService))
		})

	})

	Context("ServiceEntry cleanup", func() {

		It("should remove ServiceEntry when authorizer is removed from Istio CR", func() {
			//given
			client := createFakeClient()
			reconciler := external_authorizer.NewReconciler(client)

			//when
			err := reconciler.Reconcile(context.TODO(), istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			var s networkingv1alpha3.ServiceEntryList
			listErr := client.List(context.TODO(), &s)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(s.Items).To(HaveLen(1))

			// given
			updatedIstioCr := istioCR.DeepCopy()
			updatedIstioCr.Spec.Config = operatorv1alpha2.Config{}

			// when
			err = reconciler.Reconcile(context.TODO(), *updatedIstioCr)

			// then
			Expect(err).To(Not(HaveOccurred()))

			listErr = client.List(context.TODO(), &s)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(s.Items).To(HaveLen(0))
		})

		It("should not owned service entries", func() {
			//given
			differentNamespaceServiceEntry := networkingv1alpha3.ServiceEntry{
				ObjectMeta: metav1.ObjectMeta{Name: "test-name", Namespace: "test-namespace"},
			}

			kymaSystemNotOwnedServiceEntry := networkingv1alpha3.ServiceEntry{
				ObjectMeta: metav1.ObjectMeta{Name: "test-name", Namespace: "kyma-system"},
			}

			client := createFakeClient(&differentNamespaceServiceEntry, &kymaSystemNotOwnedServiceEntry)
			reconciler := external_authorizer.NewReconciler(client)

			updatedIstioCr := istioCR.DeepCopy()
			updatedIstioCr.Spec.Config = operatorv1alpha2.Config{}

			// when
			err := reconciler.Reconcile(context.TODO(), *updatedIstioCr)

			// then
			Expect(err).To(Not(HaveOccurred()))

			var s networkingv1alpha3.ServiceEntryList
			listErr := client.List(context.TODO(), &s)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(s.Items).To(HaveLen(2))
		})

		It("should remove owned service entries", func() {
			//given
			kymaSystemOwnedServiceEntry := networkingv1alpha3.ServiceEntry{
				ObjectMeta: metav1.ObjectMeta{Name: "test-name",
					Namespace: "kyma-system",
					Labels: map[string]string{
						"kyma-project.io/module": "istio",
					}},
			}

			client := createFakeClient(&kymaSystemOwnedServiceEntry)
			reconciler := external_authorizer.NewReconciler(client)

			updatedIstioCr := istioCR.DeepCopy()
			updatedIstioCr.Spec.Config = operatorv1alpha2.Config{}

			// when
			err := reconciler.Reconcile(context.TODO(), *updatedIstioCr)

			// then
			Expect(err).To(Not(HaveOccurred()))

			var s networkingv1alpha3.ServiceEntryList
			listErr := client.List(context.TODO(), &s)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(s.Items).To(HaveLen(0))
		})
	})
})
