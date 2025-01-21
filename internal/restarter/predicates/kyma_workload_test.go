package predicates

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Kyma Workload Predicate", func() {
	Context("RequiresProxyRestart", func() {
		It("should return true if pod in kyma-system", func() {
			predicate := KymaWorkloadRestartPredicate{}
			Expect(predicate.RequiresProxyRestart(v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "kyma-system",
				},
			})).To(BeTrue())
		})

		It("should return true if pod in has kyma label", func() {
			predicate := KymaWorkloadRestartPredicate{}
			Expect(predicate.RequiresProxyRestart(v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Labels: map[string]string{
						"kyma-project.io/module": "test",
					},
				},
			})).To(BeTrue())
		})

		It("should return false if pod in default namespace and do not have kyma label", func() {
			predicate := KymaWorkloadRestartPredicate{}
			Expect(predicate.RequiresProxyRestart(v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
				},
			})).To(BeFalse())
		})
	})
})
