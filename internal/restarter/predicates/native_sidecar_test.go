package predicates

import (
	istioCR "github.com/kyma-project/istio/operator/api/v1alpha2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Native Sidecar Predicate", func() {
	Context("Matches", func() {
		// container: regular, compatibility mode: false, annotation: not set
		It("should evaluate to true, when compatibility mode is set to false and the proxy is a regular container and nativeSidecar annotation is not set", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: false,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(false, map[string]string{})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		// container: regular, compatibility mode: false, annotation: false
		It("should evaluate to false, when compatibility mode is set to false and the proxy is a regular container and nativeSidecar annotation is set to false", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: false,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(false, map[string]string{
				nativeSidecarAnnotation: "false",
			})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		// container: regular, compatibility mode: false, annotation: true
		It("should evaluate to true, when compatibility mode is set to false and the proxy is a regular container and nativeSidecar annotation is set to true", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: false,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(false, map[string]string{
				nativeSidecarAnnotation: "true",
			})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})

		// container: regular, compatibility mode: true, annotation: not set
		It("should evaluate to false, when compatibility mode is set to true and the proxy is a regular container and nativeSidecar annotation is not set", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: true,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(false, map[string]string{})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		// container: regular, compatibility mode: true, annotation: false
		It("should evaluate to false, when compatibility mode is set to true and the proxy is a regular container and nativeSidecar annotation is set to false", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: true,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(false, map[string]string{
				nativeSidecarAnnotation: "false",
			})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		// container: regular, compatibility mode: true, annotation: true
		// should be true as annotation has priority over compatibility mode
		It("should evaluate to true, when compatibility mode is set to true and the proxy is a regular container and nativeSidecar annotation is set to true", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: true,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(false, map[string]string{
				nativeSidecarAnnotation: "true",
			})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})

		/// #######################
		/// ### INIT CONTAINERS ###
		/// #######################
		// container: initContainer, compatibility mode: false, annotation: not set
		It("should evaluate to false, when compatibility mode is set to false and the proxy is an init container and nativeSidecar annotation is not set", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: false,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(true, map[string]string{})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		// container: initContainer, compatibility mode: false, annotation: false
		It("should evaluate to true, when compatibility mode is set to false and the proxy is an init container and nativeSidecar annotation is set to false", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: false,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(true, map[string]string{
				nativeSidecarAnnotation: "false",
			})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		// container: initContainer, compatibility mode: false, annotation: true
		It("should evaluate to false, when compatibility mode is set to false and the proxy is an init container and nativeSidecar annotation is set to true", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: false,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(true, map[string]string{
				nativeSidecarAnnotation: "true",
			})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
		// container: initContainer, compatibility mode: true, annotation: not set
		It("should evaluate to true, when compatibility mode is set to true and the proxy is an init container and nativeSidecar annotation is not set", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: true,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(true, map[string]string{})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		// container: initContainer, compatibility mode: true, annotation: false
		It("should evaluate to true, when compatibility mode is set to true and the proxy is an init container and nativeSidecar annotation is set to false", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: true,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(true, map[string]string{
				nativeSidecarAnnotation: "false",
			})
			Expect(predicate.Matches(pod)).To(BeTrue())
		})
		// container: initContainer, compatibility mode: true, annotation: true
		// we don't want to restart in this case as annotation has priority over compatibility mode, and we don't want to end up in the infinite restart loop
		// so it's always should be initContainer when annotation is set to true even if compatibility mode is also true
		It("should evaluate to false, when compatibility mode is set to true and the proxy is an init container and nativeSidecar annotation is set to true", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: true,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			pod := createIstioInjectedPod(true, map[string]string{
				nativeSidecarAnnotation: "true",
			})
			Expect(predicate.Matches(pod)).To(BeFalse())
		})
	})

	Context("NewNativeSidecarRestartPredicate", func() {
		It("should create a new NativeSidecarRestartPredicate compatibility mode set to true if it's set to true in the IstioCR", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: true,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			Expect(predicate.compatibilityMode).To(BeTrue())
		})

		It("should create a new NativeSidecarRestartPredicate compatibility mode set to false if it's set to false in the IstioCR", func() {
			istioCR := &istioCR.Istio{Spec: istioCR.IstioSpec{
				CompatibilityMode: false,
			}}
			predicate := NewNativeSidecarRestartPredicate(istioCR)
			Expect(predicate.compatibilityMode).To(BeFalse())
		})
	})
})

func createIstioInjectedPod(isInitContainer bool, annotations map[string]string) v1.Pod {
	if isInitContainer {
		return v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}, Spec: v1.PodSpec{InitContainers: []v1.Container{{Name: "istio-proxy"}}}}
	}
	return v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: annotations}, Spec: v1.PodSpec{Containers: []v1.Container{{Name: "istio-proxy"}}}}
}
