package compatibility

import (
	"context"
	"github.com/kyma-project/istio/operator/pkg/labels"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Proxy Restarter", func() {
	Context("RequiresProxyRestart", func() {
		It("Should evaluate to true when proxy metadata values exist and new and old compatibility mode is different", func() {
			evaluator := ProxiesRestartEvaluator{
				oldCompatibilityMode: true,
				newCompatibilityMode: false,
				config: config{
					proxyMetadata: map[string]string{"key": "value"},
				},
			}

			Expect(evaluator.RequiresProxyRestart(v1.Pod{})).To(BeTrue())
		})

		It("Should evaluate to false when proxy metadata values exist new and old compatibility mode is equal", func() {
			evaluator := ProxiesRestartEvaluator{
				oldCompatibilityMode: true,
				newCompatibilityMode: true,
				config: config{
					proxyMetadata: map[string]string{"key": "value"},
				},
			}

			Expect(evaluator.RequiresProxyRestart(v1.Pod{})).To(BeFalse())
		})

		It("Should evaluate to false when no proxy metadata values exist new and old compatibility mode is different", func() {
			evaluator := ProxiesRestartEvaluator{
				oldCompatibilityMode: true,
				newCompatibilityMode: false,
			}

			Expect(evaluator.RequiresProxyRestart(v1.Pod{})).To(BeFalse())
		})

		It("Should evaluate to false when no proxy metadata values exist new and old compatibility mode is equal", func() {
			evaluator := ProxiesRestartEvaluator{
				oldCompatibilityMode: true,
				newCompatibilityMode: true,
			}

			Expect(evaluator.RequiresProxyRestart(v1.Pod{})).To(BeFalse())

		})
	})

	Context("NewProxyRestartEvaluator", func() {

		It("Should return an error if getLastAppliedConfiguration fails", func() {
			predicate := NewRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"compatibilityMode":abc}`,
					},
				},
			})
			_, err := predicate.NewProxyRestartEvaluator(context.Background())

			Expect(err).To(HaveOccurred())
		})

		It("Should return false for old compatibility mode if lastAppliedConfiguration is empty", func() {
			predicate := NewRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			})
			evaluator, err := predicate.NewProxyRestartEvaluator(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(evaluator).NotTo(BeNil())
			Expect(evaluator.(ProxiesRestartEvaluator).oldCompatibilityMode).To(BeFalse())
		})

		It("Should return value for old compatibility mode from lastAppliedConfiguration", func() {
			predicate := NewRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"compatibilityMode":true}`,
					},
				},
			})

			evaluator, err := predicate.NewProxyRestartEvaluator(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(evaluator).NotTo(BeNil())
			Expect(evaluator.(ProxiesRestartEvaluator).oldCompatibilityMode).To(BeTrue())
		})

		It("Should return value for new compatibility mode from istio CR", func() {
			predicate := NewRestartPredicate(&operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					CompatibilityMode: true,
				},
			})

			evaluator, err := predicate.NewProxyRestartEvaluator(context.Background())

			Expect(err).NotTo(HaveOccurred())
			Expect(evaluator).NotTo(BeNil())
			Expect(evaluator.(ProxiesRestartEvaluator).newCompatibilityMode).To(BeTrue())
		})
	})

	Context("config", func() {
		It("Should return true if proxy metadata values exist", func() {
			config := config{
				proxyMetadata: map[string]string{"key": "value"},
			}

			Expect(config.hasProxyMetadata()).To(BeTrue())
		})

		It("Should return false if proxy metadata values do not exist", func() {
			config := config{}

			Expect(config.hasProxyMetadata()).To(BeFalse())
		})
	})
})
