package istio_resources

import (
	"context"
	"strings"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	securityv1 "istio.io/client-go/pkg/apis/security/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

	It("should succeed creating peer authentication mtls", func() {
		//given
		client := createFakeClient()
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.Background(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))

		var s securityv1.PeerAuthenticationList
		listErr := client.List(context.Background(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))
	})

	It("should succeed creating config maps for dashboards", func() {
		//given
		client := createFakeClient()
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.Background(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))

		var s corev1.ConfigMapList
		listErr := client.List(context.Background(), &s)
		Expect(listErr).To(Not(HaveOccurred()))

		expectedCmNames := []string{
			"grafana-dashboard-istio-mesh",
			"grafana-dashboard-istio-performance",
			"grafana-dashboard-istio-service",
			"grafana-dashboard-istio-workload",
			"grafana-dashboard-istio-workload-performance",
		}

		for _, cm := range s.Items {
			if strings.HasPrefix("grafana-dashboard-istio", cm.Name) {
				Expect(expectedCmNames).To(ContainElement(cm.Name))
				Expect(cm.ObjectMeta.OwnerReferences).To(HaveLen(1))
				Expect(cm.ObjectMeta.OwnerReferences[0].Name).To(Equal(istioCR.Name))
				Expect(cm.ObjectMeta.OwnerReferences[0].UID).To(Equal(istioCR.UID))
			}
		}
	})

	Context("proxy-protocol EnvoyFilter", func() {
		It("should be created when hyperscaler is AWS, and ELB is to be used", func() {
			//given
			n := corev1.Node{Spec: corev1.NodeSpec{ProviderID: "aws://asdasdads"}}
			elbDeprecatedConfigMap := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "elb-deprecated",
					Namespace: "istio-system",
				},
			}
			client := createFakeClient(&n, &elbDeprecatedConfigMap)
			reconciler := NewReconciler(client)

			//when
			err := reconciler.Reconcile(context.Background(), istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(client.Get(context.Background(), ctrlclient.ObjectKey{Name: "proxy-protocol", Namespace: "istio-system"}, &networkingv1alpha3.EnvoyFilter{})).Should(Succeed())
		})

		It("should not be created when hyperscaler is AWS", func() {
			//given
			n := corev1.Node{Spec: corev1.NodeSpec{ProviderID: "aws://asdasdads"}}
			client := createFakeClient(&n)
			reconciler := NewReconciler(client)

			//when
			err := reconciler.Reconcile(context.Background(), istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			var e networkingv1alpha3.EnvoyFilter
			Expect(client.Get(context.Background(), ctrlclient.ObjectKey{Name: "proxy-protocol", Namespace: "istio-system"}, &e)).Should(Not(Succeed()))
		})

		It("should not be created when hyperscaler is not AWS", func() {
			//given
			client := createFakeClient()
			reconciler := NewReconciler(client)

			//when
			err := reconciler.Reconcile(context.Background(), istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			var e networkingv1alpha3.EnvoyFilter
			getErr := client.Get(context.Background(), ctrlclient.ObjectKey{Name: "proxy-protocol", Namespace: "istio-system"}, &e)
			Expect(getErr).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(getErr)).To(BeTrue())
		})
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
