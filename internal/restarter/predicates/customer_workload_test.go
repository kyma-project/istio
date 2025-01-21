package predicates

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Customer Workload Predicate", func() {
	Context("RequiresProxyRestart", func() {
		It("should return true if pod in default namespace", func() {
			predicate := CustomerWorkloadRestartPredicate{}
			Expect(predicate.RequiresProxyRestart(v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
			})).To(BeTrue())
		})

		It("should return false if pod in kyma-system", func() {
			predicate := CustomerWorkloadRestartPredicate{}
			Expect(predicate.RequiresProxyRestart(v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "kyma-system",
				},
			})).To(BeFalse())
		})

		It("should return false if pod has kyma label", func() {
			predicate := CustomerWorkloadRestartPredicate{}
			Expect(predicate.RequiresProxyRestart(v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Labels: map[string]string{
						"kyma-project.io/module": "test",
					},
				},
			})).To(BeFalse())
		})
	})
})
