package predicates

import (
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/pkg/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Prometheus Merge Predicate", func() {
	Context("Matches", func() {
		It("should evaluate to false when new an old prometheusMerge value is same", func() {
			predicate := PrometheusMergeRestartPredicate{
				oldPrometheusMerge: true,
				newPrometheusMerge: true,
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})
	})
	Context("NewPrometheusMergeRestartPredicate", func() {
		It("should return an error if GetLastAppliedConfiguration fails", func() {
			_, err := NewPrometheusMergeRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"telemetry":{"metrics":{"prometheusMerge":true}}}}`,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should return false for old prometheusMerge if lastAppliedConfiguration is empty", func() {
			predicate, err := NewPrometheusMergeRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldPrometheusMerge).To(BeFalse())
		})
		It("should return value for old prometheusMerge from lastAppliedConfiguration", func() {
			predicate, err := NewPrometheusMergeRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"telemetry":{"metrics":{"prometheusMerge":true}}}}`,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldPrometheusMerge).To(BeTrue())
		})
		It("should return value for new prometheusMerge from istio CR", func() {
			predicate, err := NewPrometheusMergeRestartPredicate(&operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: true,
							},
						},
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.newPrometheusMerge).To(BeTrue())
		})
		It("should return true if the new prometheusMerge is true but annotations are not updated", func() {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/port": "8080",
					},
				},
			}
			predicate, err := NewPrometheusMergeRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"telemetry":{"metrics":{"prometheusMerge":false}}}}`,
					},
				},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: true,
							},
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.Matches(pod)).To(BeTrue())

		})
		It("should return true if the new prometheusMerge is false but annotations are not updated", func() {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/path": "/stats/prometheus",
						"prometheus.io/port": "15020",
					},
				},
			}
			predicate, err := NewPrometheusMergeRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"telemetry":{"metrics":{"prometheusMerge":true}}}}`,
					},
				},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: false,
							},
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		It("should return false if the new prometheusMerge is true and annotations are correctly updated", func() {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/path": "/stats/prometheus",
						"prometheus.io/port": "15020",
					},
				},
			}
			predicate, err := NewPrometheusMergeRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"telemetry":{"metrics":{"prometheusMerge":false}}}}`,
					},
				},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: true,
							},
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		It("should return false if the new prometheusMerge is false and annotations are correctly updated", func() {
			pod := v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"prometheus.io/port": "8080",
					},
				},
			}
			predicate, err := NewPrometheusMergeRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"telemetry":{"metrics":{"prometheusMerge":true}}}}`,
					},
				},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						Telemetry: operatorv1alpha2.Telemetry{
							Metrics: operatorv1alpha2.Metrics{
								PrometheusMerge: false,
							},
						},
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
	})
})
