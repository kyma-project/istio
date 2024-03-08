package restarter_test

import (
	"context"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	"github.com/kyma-project/istio/operator/internal/restarter"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apinetworkingv1alpha3 "istio.io/api/networking/v1alpha3"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("SidecarsRestarter reconciliation", func() {
	It("should successfully restart istio ingress-gateway when it's older than EF kyma-referer", func() {
		// given
		istioCR := &operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{},
			},
		}
		kymaReferer := &v1alpha3.EnvoyFilter{
			TypeMeta: metav1.TypeMeta{
				Kind:       "EnvoyFilter",
				APIVersion: "networking.istio.io/v1alpha3",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kyma-referer",
				Namespace: gatherer.IstioNamespace,
				//has to be newer than istio-ingressgateway
				CreationTimestamp: metav1.Unix(1000, 0),
			},
			Spec: apinetworkingv1alpha3.EnvoyFilter{
				ConfigPatches: []*apinetworkingv1alpha3.EnvoyFilter_EnvoyConfigObjectPatch{
					{},
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.1")
		ingressGateway := createPodWithCreationTimestamp("istio-ingressgateway", gatherer.IstioNamespace, "discovery", "1.16.1", 100)
		fakeClient := createFakeClient(istioCR, istiod, ingressGateway, kymaReferer)
		statusHandler := status.NewStatusHandler(fakeClient)
		efReferer := istio_resources.NewEnvoyFilterAllowPartialReferer(fakeClient)
		igRestarter := restarter.NewIngressGatewayRestarter(fakeClient, []filter.IngressGatewayPredicate{efReferer}, statusHandler)

		//when
		err := igRestarter.Restart(context.Background(), istioCR)

		//then
		Expect(err).Should(Not(HaveOccurred()))
		Expect((*istioCR.Status.Conditions)[0].Reason).Should(Equal(string(operatorv1alpha2.ConditionReasonIngressGatewayReconcileSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Message).Should(Equal(operatorv1alpha2.ConditionReasonIngressGatewayReconcileSucceededMessage))
	})

	It("should fail restart of istio ingress-gateway when there is no Envoy Filter kyma-referer", func() {
		// given
		istioCR := &operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.1")
		ingressGateway := createPodWithCreationTimestamp("istio-ingressgateway", gatherer.IstioNamespace, "discovery", "1.16.1", 100)
		fakeClient := createFakeClient(istioCR, istiod, ingressGateway)
		statusHandler := status.NewStatusHandler(fakeClient)
		efReferer := istio_resources.NewEnvoyFilterAllowPartialReferer(fakeClient)
		igRestarter := restarter.NewIngressGatewayRestarter(fakeClient, []filter.IngressGatewayPredicate{efReferer}, statusHandler)

		//when
		err := igRestarter.Restart(context.Background(), istioCR)

		//then
		Expect(err).Should(HaveOccurred())
		Expect(err.Level()).To(Equal(described_errors.Error))
		Expect((*istioCR.Status.Conditions)[0].Reason).Should(Equal(string(operatorv1alpha2.ConditionReasonIngressGatewayReconcileFailed)))
		Expect((*istioCR.Status.Conditions)[0].Message).Should(Equal(operatorv1alpha2.ConditionReasonIngressGatewayReconcileFailedMessage))
	})
})
