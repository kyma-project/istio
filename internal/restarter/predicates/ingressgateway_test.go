package predicates_test

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/pkg/labels"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	predicates "github.com/kyma-project/istio/operator/internal/restarter/predicates"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/utils/ptr"
)

var _ = Describe("Ingress Gateway Predicate", func() {
	Context("RequiresIngressGatewayRestart", func() {
		It("should evaluate to true if new is nil and old is not nil", func() {
			evaluator := predicates.NumTrustedProxiesRestartEvaluator{
				NewNumTrustedProxies: nil,
				OldNumTrustedProxies: new(int),
			}

			Expect(evaluator.RequiresIngressGatewayRestart()).To(BeTrue())
		})

		It("should evaluate to true if new is not nil and old is nil", func() {
			evaluator := predicates.NumTrustedProxiesRestartEvaluator{
				NewNumTrustedProxies: new(int),
				OldNumTrustedProxies: nil,
			}

			Expect(evaluator.RequiresIngressGatewayRestart()).To(BeTrue())
		})

		It("should evaluate to true if numTrustedProxies is different", func() {
			newNumTrustedProxies := 1
			oldNumTrustedProxies := 2

			evaluator := predicates.NumTrustedProxiesRestartEvaluator{
				NewNumTrustedProxies: &newNumTrustedProxies,
				OldNumTrustedProxies: &oldNumTrustedProxies,
			}

			Expect(evaluator.RequiresIngressGatewayRestart()).To(BeTrue())
		})

		It("should evaluate to false if numTrustedProxies is the same", func() {
			numTrustedProxies := 1
			oldNumTrustedProxies := 1

			evaluator := predicates.NumTrustedProxiesRestartEvaluator{
				NewNumTrustedProxies: &oldNumTrustedProxies,
				OldNumTrustedProxies: &numTrustedProxies,
			}

			Expect(evaluator.RequiresIngressGatewayRestart()).To(BeFalse())

		})
	})

	Context("NewIngressGatewayEvaluator", func() {
		const (
			mockIstioTag             string = "1.16.1-distroless"
			lastAppliedConfiguration        = labels.LastAppliedConfiguration
		)

		It("should return an error if getLastAppliedConfiguration fails", func() {
			predicate := predicates.NewIngressGatewayRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						lastAppliedConfiguration: `{"config":{"numTrustedProxies":abc},"IstioTag":w}`,
					},
				},
			})
			_, err := predicate.NewIngressGatewayEvaluator(context.Background())

			Expect(err).To(HaveOccurred())
		})

		It("should return nil for old numTrustedProxies if lastAppliedConfiguration is empty", func() {
			predicate := predicates.NewIngressGatewayRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			})
			evaluator, err := predicate.NewIngressGatewayEvaluator(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(evaluator).NotTo(BeNil())
			Expect(evaluator.(predicates.NumTrustedProxiesRestartEvaluator).OldNumTrustedProxies).To(BeNil())
		})

		It("should return correct not nil value for new and old numTrustedProxies", func() {
			istio := &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						NumTrustedProxies: ptr.To(1),
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{lastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":2},"IstioTag":"%s"}`, mockIstioTag)},
				},
			}

			predicate := predicates.NewIngressGatewayRestartPredicate(istio)
			evaluator, err := predicate.NewIngressGatewayEvaluator(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(evaluator).NotTo(BeNil())
			Expect(*(evaluator.(predicates.NumTrustedProxiesRestartEvaluator).NewNumTrustedProxies)).To(Equal(1))
			Expect(*(evaluator.(predicates.NumTrustedProxiesRestartEvaluator).OldNumTrustedProxies)).To(Equal(2))
		})

		It("should return correct nil value for new and old numTrustedProxies", func() {
			istio := &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						NumTrustedProxies: nil,
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{lastAppliedConfiguration: fmt.Sprintf(`{"IstioTag":"%s"}`, mockIstioTag)},
				},
			}

			predicate := predicates.NewIngressGatewayRestartPredicate(istio)
			evaluator, err := predicate.NewIngressGatewayEvaluator(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(evaluator).NotTo(BeNil())
			Expect(evaluator.(predicates.NumTrustedProxiesRestartEvaluator).NewNumTrustedProxies).To(BeNil())
			Expect(evaluator.(predicates.NumTrustedProxiesRestartEvaluator).OldNumTrustedProxies).To(BeNil())
		})

	})
})
