package predicates

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Native Sidecar Predicate", func() {
	Context("Matches", func() {
		// container: regular, annotation: not set
		It("should evaluate to true, when proxy is a regular container and nativeSidecar annotation is not set", func() {
			predicate := NewNativeSidecarRestartPredicate()
			pod := createIstioInjectedPod(false, map[string]string{})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		// container: regular, annotation: false
		It("should evaluate to false, when proxy is a regular container and nativeSidecar annotation is set to false", func() {
			predicate := NewNativeSidecarRestartPredicate()
			pod := createIstioInjectedPod(false, map[string]string{
				nativeSidecarAnnotation: "false",
			})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		// container: regular, annotation: true
		It("should evaluate to true, when proxy is a regular container and nativeSidecar annotation is set to true", func() {
			predicate := NewNativeSidecarRestartPredicate()
			pod := createIstioInjectedPod(false, map[string]string{
				nativeSidecarAnnotation: "true",
			})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})

		/// #######################
		/// ### INIT CONTAINERS ###
		/// #######################
		// container: initContainer, annotation: not set
		It("should evaluate to false, when proxy is an init container and nativeSidecar annotation is not set", func() {
			predicate := NewNativeSidecarRestartPredicate()
			pod := createIstioInjectedPod(true, map[string]string{})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		// container: initContainer, annotation: false
		It("should evaluate to true, when proxy is an init container and nativeSidecar annotation is set to false", func() {
			predicate := NewNativeSidecarRestartPredicate()
			pod := createIstioInjectedPod(true, map[string]string{
				nativeSidecarAnnotation: "false",
			})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		// container: initContainer, annotation: true
		It("should evaluate to false, when proxy is an init container and nativeSidecar annotation is set to true", func() {
			predicate := NewNativeSidecarRestartPredicate()
			pod := createIstioInjectedPod(true, map[string]string{
				nativeSidecarAnnotation: "true",
			})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
	})
})

func createIstioInjectedPod(isInitContainer bool, annotations map[string]string) v1.Pod {
	if isInitContainer {
		return v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}, Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: "istio-proxy"}}}}
	}
	return v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "istio-proxy"}}}}
}
