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
		It("should evaluate to true when new and old enablePrometheusMerge value is different", func() {
			predicate := ProxyRestartPredicate{
				oldEnablePrometheusMerge: true,
				newEnablePrometheusMerge: false,
			}
			Expect(predicate.RequiresProxyRestart(v1.Pod{})).To(BeTrue())
		})
		It("should evaluate to false when new an old enablePrometheusMerge value is same", func() {
			predicate := ProxyRestartPredicate{
				oldEnablePrometheusMerge: true,
				newEnablePrometheusMerge: false,
			}
			Expect(predicate.RequiresProxyRestart(v1.Pod{})).To(BeTrue())
		})
	})
	Context("NewRestartPredicate", func() {
		It("should return an error if GetLastAppliedConfiguration fails", func() {
			_, err := NewRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"enablePrometheusMerge":true}}`,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())
		})
		It("should return false for old enablePrometheusMerge if lastAppliedConfiguration is empty", func() {
			predicate, err := NewRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldEnablePrometheusMerge).To(BeFalse())
		})
		It("should return value for old enablePrometheusMerge from lastAppliedConfiguration", func() {
			predicate, err := NewRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"enablePrometheusMerge":true}}`,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldEnablePrometheusMerge).To(BeTrue())
		})
		It("should return value for new enablePrometheusMerge from istio CR", func() {
			predicate, err := NewRestartPredicate(&operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						EnablePrometheusMerge: true,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.newEnablePrometheusMerge).To(BeTrue())
		})
	})
})
