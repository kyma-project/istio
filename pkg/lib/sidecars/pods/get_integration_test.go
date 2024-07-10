package pods_test

import (
	"context"

	. "github.com/onsi/gomega"

	. "github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Pods", Serial, func() {
	Context("GetPodsToRestart", func() {
		It("Should respect defined limit when getting pods to restart", func() {
			// given
			pod := createPod("test-pod")
			Expect(k8sClient.Create(context.Background(), pod)).Should(Succeed())

			// Eventually(func(g Gomega) {
			// 	err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(pod), pod)
			// 	g.Expect(err).To(Not(HaveOccurred()))
			// 	fmt.Printf("Pod: %v\n", pod.Status)
			// 	g.Expect(pod.Status.Phase).To(Equal(v1.PodRunning))
			// }, eventuallyTimeout).Should(Succeed())

			// // when
			// expectedImage := pods.NewSidecarImage("europe-docker.pkg.dev/kyma-project/prod/external/istio", "1.22.2-distroless")
			// podsToRestart, err := pods.GetPodsToRestart(context.Background(), k8sClient, expectedImage, helpers.DefaultSidecarResources, []filter.SidecarProxyPredicate{}, 1, &logr.Logger{})

			// // then
			// Expect(err).ToNot(HaveOccurred())
			// Expect(podsToRestart.Items).To(HaveLen(1))

			// // cleanup
			// Expect(k8sClient.Delete(context.Background(), pod)).Should(Succeed())
		})
	})
})

func createPod(name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: testNamespace,
			Annotations: map[string]string{
				"sidecar.istio.io/status": "injected",
			},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "httpbin",
					Image: "docker.io/kennethreitz/httpbin",
				},
				{
					Name:  "istio-proxy",
					Image: "europe-docker.pkg.dev/kyma-project/prod/external/istio/proxyv2:1.22.1-distroless",
					Resources: v1.ResourceRequirements{
						Limits: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("100m"),
							v1.ResourceMemory: resource.MustParse("200Mi"),
						},
						Requests: v1.ResourceList{
							v1.ResourceCPU:    resource.MustParse("50m"),
							v1.ResourceMemory: resource.MustParse("100Mi"),
						},
					},
				},
			},
		},
	}
}
