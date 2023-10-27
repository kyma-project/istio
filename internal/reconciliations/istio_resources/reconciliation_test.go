package istio_resources

import (
	"context"
	"fmt"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	networkingv1beta1 "istio.io/client-go/pkg/apis/networking/v1beta1"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"strings"
)

var _ = Describe("Reconciliation", func() {
	numTrustedProxies := 1
	istioCR := operatorv1alpha1.Istio{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
			UID:             "1234",
		},
		Spec: operatorv1alpha1.IstioSpec{
			Config: operatorv1alpha1.Config{
				NumTrustedProxies: &numTrustedProxies,
			},
		},
	}

	It("should succeed creating envoy filter referer", func() {
		client := createFakeClient()
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.TODO(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))

		var s networkingv1alpha3.EnvoyFilterList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))
	})

	It("should succeed creating gateway kyma", func() {
		client := createFakeClient()
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.TODO(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))

		var s networkingv1alpha3.GatewayList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))
		Expect(s.Items[0].Spec.Servers[0].Hosts[0]).To(Equal(fmt.Sprintf("*.%s", clusterconfig.LocalKymaDomain)))
	})

	It("should succeed creating virtual service healthz", func() {
		client := createFakeClient()
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.TODO(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))

		var s networkingv1beta1.VirtualServiceList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))
		Expect(s.Items[0].Spec.Hosts[0]).To(Equal(fmt.Sprintf("healthz.%s", clusterconfig.LocalKymaDomain)))
	})

	It("should succeed creating peer authentication mtls", func() {
		client := createFakeClient()
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.TODO(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))

		var s securityv1beta1.PeerAuthenticationList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))
	})

	It("should succeed creating config maps for dashboards", func() {
		client := createFakeClient()
		reconciler := NewReconciler(client)

		//when
		err := reconciler.Reconcile(context.TODO(), istioCR)

		//then
		Expect(err).To(Not(HaveOccurred()))

		var s corev1.ConfigMapList
		listErr := client.List(context.TODO(), &s)
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

		It("should be created when hyperscaler is AWS", func() {
			client := createFakeClient(createAwsShootInfo())
			reconciler := NewReconciler(client)

			//when
			err := reconciler.Reconcile(context.TODO(), istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			var e networkingv1alpha3.EnvoyFilter
			Expect(client.Get(context.TODO(), ctrlclient.ObjectKey{Name: "proxy-protocol", Namespace: "istio-system"}, &e)).Should(Succeed())
		})

		It("should not be created when hyperscaler is not AWS", func() {
			client := createFakeClient(createGcpShootInfo())
			reconciler := NewReconciler(client)

			//when
			err := reconciler.Reconcile(context.TODO(), istioCR)

			//then
			Expect(err).To(Not(HaveOccurred()))

			var e networkingv1alpha3.EnvoyFilter
			getErr := client.Get(context.TODO(), ctrlclient.ObjectKey{Name: "proxy-protocol", Namespace: "istio-system"}, &e)
			Expect(getErr).To(HaveOccurred())
			Expect(k8serrors.IsNotFound(getErr)).To(BeTrue())
		})
	})

})

func createFakeClient(objects ...ctrlclient.Object) ctrlclient.Client {
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

func createGcpShootInfo() *corev1.ConfigMap {
	return createShootInfoConfigMap("gcp")
}

func createAwsShootInfo() *corev1.ConfigMap {
	return createShootInfoConfigMap("aws")
}

func createShootInfoConfigMap(provider string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: clusterconfig.ConfigMapShootInfoName, Namespace: clusterconfig.ConfigMapShootInfoNS},
		Data:       map[string]string{"provider": provider},
	}
}
