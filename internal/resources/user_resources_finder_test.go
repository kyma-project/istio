package resources

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"istio.io/api/networking/v1alpha3"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("IstioResourceFinder - UserCreated EnvoyFilters", func() {
	It("Should return nil if there are no user's EnvoyFilters targeting istio-ingressgateway", func() {
		k8sClient := fake.NewClientBuilder().WithScheme(sc).WithObjects(&networkingv1alpha3.EnvoyFilter{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "istio-system",
				Name:      "ef-from-rate-limit",
				OwnerReferences: []metav1.OwnerReference{
					{
						Kind: "RateLimit",
					},
				},
			},
			// some EnvoyFilter are present, but they are owned by APIGateway RateLimit
		}, &networkingv1alpha3.EnvoyFilter{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "other-system",
				Name:      "not-targeting-ingressgateway2",
			},
		}).Build()

		urf := NewUserResources(k8sClient)

		err := urf.DetectUserCreatedEfOnIngress(context.Background())

		Expect(err).To(Not(HaveOccurred()))
	})

	It("Should return described error if there are present user-created EnvoyFilters in the cluster", func() {
		const efName = "ef-targeting-ingressgateway"
		const efNamespace = "istio-system"
		k8sClient := fake.NewClientBuilder().WithScheme(sc).WithObjects(
			&networkingv1alpha3.EnvoyFilter{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: efNamespace,
					Name:      efName,
				},
				Spec: v1alpha3.EnvoyFilter{WorkloadSelector: &v1alpha3.WorkloadSelector{Labels: map[string]string{"app": "istio-ingressgateway"}}},
			}).Build()

		urf := NewUserResources(k8sClient)

		err := urf.DetectUserCreatedEfOnIngress(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(err.Level()).To(Equal(described_errors.Warning))
		Expect(err.Error()).To(Equal(fmt.Sprintf("user-created EnvoyFilter %s/%s targeting Ingress Gateway found", efNamespace, efName)))
		Expect(err.Description()).To(Equal(fmt.Sprintf("misconfigured EnvoyFilter can potentially break Istio Ingress Gateway: user-created EnvoyFilter %s/%s targeting Ingress Gateway found", efNamespace, efName)))
	})
})
