package ingressgateway_test

import (
	"context"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	ingressgateway "github.com/kyma-project/istio/operator/pkg/lib/ingress_gateway"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Ingress Gateway Restarter", func() {
	Context("RequiresIngressGatewayRestart", func() {
		It("Should evaluate to true if new is nil and old is nil", func() {
			evaluator := ingressgateway.IngressGatewayRestartEvaluator{
				newNumTrustedProxies: nil,
				oldNumTrustedProxies: new(int),
			}

			Expect(evaluator.RequiresIngressGatewayRestart()).To(BeTrue())
		})

		It("Should evaluate to true if new is not nil and old is nil", func() {
			evaluator := ingressgateway.IngressGatewayRestartEvaluator{
				newNumTrustedProxies: new(int),
				oldNumTrustedProxies: nil,
			}

			Expect(evaluator.RequiresIngressGatewayRestart()).To(BeTrue())
		})

		It("Should evaluate to true if numTrustedProxies is different", func() {
			newNumTrustedProxies := 1
			oldNumTrustedProxies := 2

			evaluator := ingressgateway.IngressGatewayRestartEvaluator{
				newNumTrustedProxies: &newNumTrustedProxies,
				oldNumTrustedProxies: &oldNumTrustedProxies,
			}

			Expect(evaluator.RequiresIngressGatewayRestart()).To(BeTrue())
		})
	})

	Context("NewIngressGatewayEvaluator", func() {

		It("Should return an error if getLastAppliedConfiguration fails", func() {
			predicate := ingressgateway NewIngressGatewayRestartPredicate(&operatorv1alpha2.Istio{})
			_, err := predicate.NewIngressGatewayEvaluator(context.Background())

			Expect(err).To(HaveOccurred())
		})
	})
})
