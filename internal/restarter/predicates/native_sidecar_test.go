package predicates

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Native Sidecar Predicate", func() {
	DescribeTable("Matches",
		func(isInitContainer bool, annotations map[string]string, expectedResult bool) {
			predicate := NewNativeSidecarRestartPredicate()
			pod := createIstioInjectedPod(isInitContainer, annotations)
			Expect(predicate.Matches(pod)).To(Equal(expectedResult))
		},
		// container: regular, annotation: not set
		Entry("should evaluate to true, when proxy is a regular container and nativeSidecar annotation is not set",
			false, map[string]string{}, true),
		// container: regular, annotation: false
		Entry("should evaluate to false, when proxy is a regular container and nativeSidecar annotation is set to false",
			false, map[string]string{nativeSidecarAnnotation: "false"}, false),
		// container: regular, annotation: true
		Entry("should evaluate to true, when proxy is a regular container and nativeSidecar annotation is set to true",
			false, map[string]string{nativeSidecarAnnotation: "true"}, true),
		// container: initContainer, annotation: not set
		Entry("should evaluate to false, when proxy is an init container and nativeSidecar annotation is not set",
			true, map[string]string{}, false),
		// container: initContainer, annotation: false
		Entry("should evaluate to true, when proxy is an init container and nativeSidecar annotation is set to false",
			true, map[string]string{nativeSidecarAnnotation: "false"}, true),
		// container: initContainer, annotation: true
		Entry("should evaluate to false, when proxy is an init container and nativeSidecar annotation is set to true",
			true, map[string]string{nativeSidecarAnnotation: "true"}, false),
	)
})

func createIstioInjectedPod(isInitContainer bool, annotations map[string]string) v1.Pod {
	if isInitContainer {
		return v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}, Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: "istio-proxy"}}}}
	}
	return v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "istio-proxy"}}}}
}
