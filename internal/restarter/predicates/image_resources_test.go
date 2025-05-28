package predicates_test

import (
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Matches", func() {
	It("should should return false when pod has custom image annotation", func() {
		// given
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", map[string]string{"sidecar.istio.io/proxyImage": "istio/proxyv2:1.21.0"}, false)
		predicate := predicates.NewImageResourcesPredicate(predicates.NewSidecarImage("istio", "1.22.0"), v1.ResourceRequirements{})

		// when
		shouldRestart := predicate.Matches(pod)

		// then
		Expect(shouldRestart).To(BeFalse())
	})

	It("should return true when pod does not have custom image annotation", func() {
		// given
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", map[string]string{}, false)
		predicate := predicates.NewImageResourcesPredicate(predicates.NewSidecarImage("istio", "1.22.0"), v1.ResourceRequirements{})

		// when
		shouldRestart := predicate.Matches(pod)

		// then
		Expect(shouldRestart).To(BeTrue())
	})

})

var _ = Describe("handling of proxy sidecar as container or init container", func() {
	It("should return true when sidecar is a regular container in the Pod", func() {
		// given
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", map[string]string{}, false)
		predicate := predicates.NewImageResourcesPredicate(predicates.NewSidecarImage("istio", "1.22.0"), v1.ResourceRequirements{})

		// when
		shouldRestart := predicate.Matches(pod)

		// then
		Expect(shouldRestart).To(BeTrue())
	})

	It("should return true when sidecar is an init container in the Pod", func() {
		// given
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", map[string]string{}, true)
		predicate := predicates.NewImageResourcesPredicate(predicates.NewSidecarImage("istio", "1.22.0"), v1.ResourceRequirements{})

		// when
		shouldRestart := predicate.Matches(pod)

		// then
		Expect(shouldRestart).To(BeTrue())
	})
})

var _ = Describe("IsReadyWithIstioAnnotation", func() {
	It("should return true when pod is ready and has istio sidecar status annotation", func() {
		// given
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", map[string]string{"sidecar.istio.io/status": "true"}, false)

		// when
		isReady := predicates.IsReadyWithIstioAnnotation(pod)

		// then
		Expect(isReady).To(BeTrue())
	})

	It("should return false when pod is not ready", func() {
		// given
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", map[string]string{"sidecar.istio.io/status": "true"}, false)
		pod.Status.Conditions[0].Status = v1.ConditionFalse

		// when
		isReady := predicates.IsReadyWithIstioAnnotation(pod)

		// then
		Expect(isReady).To(BeFalse())
	})

	It("should return false when pod does not have istio sidecar status annotation", func() {
		// given
		pod := createPodWithProxySidecar("test-pod", "test-namespace", "1.21.0", nil, false)
		delete(pod.Annotations, "sidecar.istio.io/status")

		// when
		isReady := predicates.IsReadyWithIstioAnnotation(pod)

		// then
		Expect(isReady).To(BeFalse())
	})
})

func createPodWithProxySidecar(name, namespace, proxyIstioVersion string, annotations map[string]string, proxyInitContainer bool) v1.Pod {
	if annotations == nil {
		annotations = map[string]string{}
	}
	var spec v1.PodSpec
	if proxyInitContainer {
		spec = v1.PodSpec{
			InitContainers: []v1.Container{
				{
					Name:  "istio-proxy",
					Image: "istio/proxyv2:" + proxyIstioVersion,
				},
			},
		}
	} else {
		spec = v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  "istio-proxy",
					Image: "istio/proxyv2:" + proxyIstioVersion,
				},
			},
		}
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
		Spec: spec,
	}
}
