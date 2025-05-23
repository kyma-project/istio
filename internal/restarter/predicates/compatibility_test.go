package predicates

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/pkg/labels"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
)

var _ = Describe("Compatibility Predicate", func() {
	Context("Matches", func() {
		It("should evaluate to true when proxy metadata values exist and new and old compatibility mode is different", func() {
			predicate := CompatibilityRestartPredicate{
				oldCompatibilityMode: true,
				newCompatibilityMode: false,
				config: config{
					proxyMetadata: map[string]string{"key": "value"},
				},
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})

		It("should evaluate to false when proxy metadata values exist and new and old compatibility mode is equal", func() {
			predicate := CompatibilityRestartPredicate{
				oldCompatibilityMode: true,
				newCompatibilityMode: true,
				config: config{
					proxyMetadata: map[string]string{"key": "value"},
				},
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})

		It("should evaluate to false when no proxy metadata values exist and new and old compatibility mode is different", func() {
			predicate := CompatibilityRestartPredicate{
				oldCompatibilityMode: true,
				newCompatibilityMode: false,
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})

		It("should evaluate to false when no proxy metadata values exist and new and old compatibility mode is equal", func() {
			predicate := CompatibilityRestartPredicate{
				oldCompatibilityMode: true,
				newCompatibilityMode: true,
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})
	})

	Context("NewCompatibilityRestartPredicate", func() {
		It("should return an error if getLastAppliedConfiguration fails", func() {
			_, err := NewCompatibilityRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"compatibilityMode":abc}`,
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("should return false for old compatibility mode if lastAppliedConfiguration is empty", func() {
			predicate, err := NewCompatibilityRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldCompatibilityMode).To(BeFalse())
		})

		It("should return value for old compatibility mode from lastAppliedConfiguration", func() {
			predicate, err := NewCompatibilityRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"compatibilityMode":true}`,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldCompatibilityMode).To(BeTrue())
		})

		It("should return value for new compatibility mode from istio CR", func() {
			predicate, err := NewCompatibilityRestartPredicate(&operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					CompatibilityMode: true,
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.newCompatibilityMode).To(BeTrue())
		})
	})

	Context("config", func() {
		It("should return true if proxy metadata values exist", func() {
			config := config{
				proxyMetadata: map[string]string{"key": "value"},
			}

			Expect(config.hasProxyMetadata()).To(BeTrue())
		})

		It("should return false if proxy metadata values do not exist", func() {
			config := config{}
			Expect(config.hasProxyMetadata()).To(BeFalse())
		})
	})
})
