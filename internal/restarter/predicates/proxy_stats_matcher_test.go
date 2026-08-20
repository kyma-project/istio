package predicates

import (
	"github.com/kyma-project/istio/operator/pkg/labels"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio/configuration"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ProxyStatsMatcher Predicate", func() {
	Context("Matches", func() {
		It("should evaluate to false if inclusionRegexps are the same", func() {
			predicate := ProxyStatsMatcherRestartPredicate{
				oldInclusionRegexps: []string{".*upstream_rq_retry.*"},
				newInclusionRegexps: []string{".*upstream_rq_retry.*"},
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})

		It("should evaluate to false if both inclusionRegexps are nil", func() {
			predicate := ProxyStatsMatcherRestartPredicate{
				oldInclusionRegexps: nil,
				newInclusionRegexps: nil,
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})

		It("should evaluate to false if one inclusionRegexps is nil and the other is empty", func() {
			predicate := ProxyStatsMatcherRestartPredicate{
				oldInclusionRegexps: nil,
				newInclusionRegexps: []string{},
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})

		It("should evaluate to true if inclusionRegexps values differ", func() {
			predicate := ProxyStatsMatcherRestartPredicate{
				oldInclusionRegexps: []string{".*upstream_rq_retry.*"},
				newInclusionRegexps: []string{".*upstream_cx.*"},
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})

		It("should evaluate to true if a regexp is added", func() {
			predicate := ProxyStatsMatcherRestartPredicate{
				oldInclusionRegexps: []string{".*upstream_rq_retry.*"},
				newInclusionRegexps: []string{".*upstream_rq_retry.*", ".*upstream_cx.*"},
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})

		It("should evaluate to true if a regexp is removed", func() {
			predicate := ProxyStatsMatcherRestartPredicate{
				oldInclusionRegexps: []string{".*upstream_rq_retry.*", ".*upstream_cx.*"},
				newInclusionRegexps: []string{".*upstream_rq_retry.*"},
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})

		It("should evaluate to true if old inclusionRegexps is nil and new is not", func() {
			predicate := ProxyStatsMatcherRestartPredicate{
				oldInclusionRegexps: nil,
				newInclusionRegexps: []string{".*upstream_rq_retry.*"},
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})

		It("should evaluate to true if new inclusionRegexps is nil and old is not", func() {
			predicate := ProxyStatsMatcherRestartPredicate{
				oldInclusionRegexps: []string{".*upstream_rq_retry.*"},
				newInclusionRegexps: nil,
			}
			Expect(predicate.Matches(v1.Pod{})).To(BeTrue())
		})
	})

	Context("NewProxyStatsMatcherRestartPredicate", func() {
		It("should evaluate to false if inclusionRegexps are the same but in different order", func() {
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"proxyStatsMatcher":{"inclusionRegexps":[".*upstream_rq_retry.*",".*upstream_cx.*"]}}}`,
					},
				},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						ProxyStatsMatcher: &operatorv1alpha2.ProxyStatsMatcher{
							InclusionRegexps: []string{".*upstream_cx.*", ".*upstream_rq_retry.*"},
						},
					},
				},
			}
			lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
			Expect(err).NotTo(HaveOccurred())

			predicate := NewProxyStatsMatcherRestartPredicate(istioCR, lastAppliedConfig)
			Expect(predicate.Matches(v1.Pod{})).To(BeFalse())
		})

		It("should return nil for oldInclusionRegexps if lastAppliedConfiguration is empty", func() {
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{}},
			}
			lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
			Expect(err).NotTo(HaveOccurred())

			predicate := NewProxyStatsMatcherRestartPredicate(istioCR, lastAppliedConfig)
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldInclusionRegexps).To(BeNil())
		})

		It("should return nil for oldInclusionRegexps if proxyStatsMatcher is absent from lastAppliedConfiguration", func() {
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"enableDNSProxying":true}}`,
					},
				},
			}
			lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
			Expect(err).NotTo(HaveOccurred())

			predicate := NewProxyStatsMatcherRestartPredicate(istioCR, lastAppliedConfig)
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldInclusionRegexps).To(BeNil())
		})

		It("should return inclusionRegexps from lastAppliedConfiguration", func() {
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						labels.LastAppliedConfiguration: `{"config":{"proxyStatsMatcher":{"inclusionRegexps":[".*upstream_rq_retry.*",".*upstream_cx.*"]}}}`,
					},
				},
			}
			lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
			Expect(err).NotTo(HaveOccurred())

			predicate := NewProxyStatsMatcherRestartPredicate(istioCR, lastAppliedConfig)
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.oldInclusionRegexps).To(ConsistOf(".*upstream_rq_retry.*", ".*upstream_cx.*"))
		})

		It("should return inclusionRegexps from the Istio CR spec", func() {
			istioCR := &operatorv1alpha2.Istio{
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						ProxyStatsMatcher: &operatorv1alpha2.ProxyStatsMatcher{
							InclusionRegexps: []string{".*upstream_rq_retry.*"},
						},
					},
				},
			}
			lastAppliedConfig, err := configuration.GetLastAppliedConfiguration(istioCR)
			Expect(err).NotTo(HaveOccurred())

			predicate := NewProxyStatsMatcherRestartPredicate(istioCR, lastAppliedConfig)
			Expect(predicate).NotTo(BeNil())
			Expect(predicate.newInclusionRegexps).To(ConsistOf(".*upstream_rq_retry.*"))
		})
	})
})
