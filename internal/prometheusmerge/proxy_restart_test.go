package prometheusmerge

import (
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/pkg/labels"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Proxy Restarter", func() {
	Context("RequiresProxyRestart", func() {
		It("should evaluate to true when new and old prometheusMerge value is different", func() {
			predicate := ProxyRestartPredicate{
				oldPrometheusMerge: true,
				newPrometheusMerge: false,
			}
			Expect(predicate.RequiresProxyRestart(v1.Pod{})).To(BeTrue())
		})
		It("should evaluate to false when new an old prometheusMerge value is same", func() {
			predicate := ProxyRestartPredicate{
				oldPrometheusMerge: true,
				newPrometheusMerge: false,
			}
			Expect(predicate.RequiresProxyRestart(v1.Pod{})).To(BeTrue())
		})
	})
	Context("NewRestartPredicate", func() {
		It("should return an error if GetLastAppliedConfiguration fails", func() {
			_, err := NewRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"telemetry":{"metrics":{"prometheusMerge":true}}}}`,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should return false for old prometheusMerge if lastAppliedConfiguration is empty", func() {
			predicate, err := NewRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldPrometheusMerge).To(BeFalse())
		})
		It("should return value for old prometheusMerge from lastAppliedConfiguration", func() {
			predicate, err := NewRestartPredicate(&operatorv1alpha2.Istio{
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
			predicate, err := NewRestartPredicate(&operatorv1alpha2.Istio{
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
	})
})
