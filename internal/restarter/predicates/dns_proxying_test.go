package predicates

import (
	"github.com/kyma-project/istio/operator/pkg/labels"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("EnableDNSProxying Predicate", func() {
	Context("Matches", func() {
		It("should evaluate to false if newEnableDNSProxying is the same as oldEnableDNSProxying", func() {
			predicate := EnableDNSProxyingRestartPredicate{
				oldEnableDNSProxying: ptr.To(true),
				newEnableDNSProxying: ptr.To(true),
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})

		It("should evaluate to true if newEnableDNSProxying is different from oldEnableDNSProxying", func() {
			predicate := EnableDNSProxyingRestartPredicate{
				oldEnableDNSProxying: ptr.To(true),
				newEnableDNSProxying: ptr.To(false),
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})

		It("should evaluate to true if newEnableDNSProxying is nil and oldEnableDNSProxying is not nil", func() {
			predicate := EnableDNSProxyingRestartPredicate{
				oldEnableDNSProxying: ptr.To(true),
				newEnableDNSProxying: nil,
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})

		It("should evaluate to true if newEnableDNSProxying is not nil and oldEnableDNSProxying is nil", func() {
			predicate := EnableDNSProxyingRestartPredicate{
				oldEnableDNSProxying: nil,
				newEnableDNSProxying: ptr.To(false),
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})
	})
	//todo:
	Context("NewEnableDNSProxyingRestartPredicate", func() {
		It("should return an error if GetLastAppliedConfiguration fails", func() {
			_, err := NewEnableDNSProxyingRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"erroneous-annotation-mock":erroneous-annotation-mock}`,
					},
				},
			})
			Expect(err).To(HaveOccurred())
		})

		It("should return nil for oldEnableDNSProxying if lastAppliedConfiguration is empty", func() {
			predicate, err := NewEnableDNSProxyingRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldEnableDNSProxying).To(BeNil())
		})

		It("should return value for oldEnableDNSProxying from lastAppliedConfiguration", func() {
			predicate, err := NewEnableDNSProxyingRestartPredicate(&operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"enableDNSProxying":true}}`,
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(*predicate.oldEnableDNSProxying).To(BeTrue())
		})

		It("should return value for newEnableDNSProxying from istio CR", func() {
			predicate, err := NewEnableDNSProxyingRestartPredicate(&operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						EnableDNSProxying: ptr.To(true),
					},
				},
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(predicate).NotTo(BeNil())
			Expect(*predicate.newEnableDNSProxying).To(BeTrue())
		})
	})
})
