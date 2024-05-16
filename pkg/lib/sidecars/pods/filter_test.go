package pods_test

import (
	"context"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Evaluate restart", func() {
	It("should should return false when pod has custom image annotation", func() {
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", map[string]string{"sidecar.istio.io/proxyImage": "istio/proxyv2:1.22.0"})

		predicate := pods.NewRestartProxyPredicate(pods.NewSidecarImage("istio", "1.22.0"), v1.ResourceRequirements{})
		evaluator, err := predicate.NewProxyRestartEvaluator(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(evaluator.RequiresProxyRestart(pod)).To(BeFalse())

	})

	It("should should return true when pod does not have custom image annotation", func() {
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", map[string]string{})

		predicate := pods.NewRestartProxyPredicate(pods.NewSidecarImage("istio", "1.22.0"), v1.ResourceRequirements{})
		evaluator, err := predicate.NewProxyRestartEvaluator(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(evaluator.RequiresProxyRestart(pod)).To(BeTrue())

	})
})

func createPodWithProxySidecar(name, namespace, proxyIstioVersion string, annotations map[string]string) v1.Pod {
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["sidecar.istio.io/status"] = "true"
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
		Status: v1.PodStatus{
			Phase: v1.PodRunning,
			Conditions: []v1.PodCondition{
				{
					Type:   v1.PodReady,
					Status: v1.ConditionTrue,
				},
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "istio-proxy",
					Image: "istio/proxyv2:" + proxyIstioVersion,
				},
			},
		},
	}
}
