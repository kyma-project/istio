package istio_resources

import (
	"context"
	"time"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Apply", func() {
	templateValues := map[string]string{}
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	It("should return created and annotate with timestamp if no resource was present", func() {
		client := createFakeClient()
		sample := NewEnvoyFilterAllowPartialReferer(client)

		//when
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		var s networkingv1alpha3.EnvoyFilterList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[EnvoyFilterAnnotation]).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[istio.DisclaimerKey]).To(Not(BeNil()))
	})

	It("should return not changed and annotate with timestamp if no change is needed", func() {
		//given
		var filter networkingv1alpha3.EnvoyFilter
		err := yaml.Unmarshal(manifest_ef_allow_partial_referer, &filter)
		Expect(err).To(Not(HaveOccurred()))

		client := createFakeClient(&filter)

		sample := NewEnvoyFilterAllowPartialReferer(client)

		//when
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultNone))

		var s networkingv1alpha3.EnvoyFilterList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[EnvoyFilterAnnotation]).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[istio.DisclaimerKey]).To(Not(BeNil()))
	})

	It("should return updated and annotate with timestamp if change is needed", func() {
		//given
		var filter networkingv1alpha3.EnvoyFilter
		err := yaml.Unmarshal(manifest_ef_allow_partial_referer, &filter)
		Expect(err).To(Not(HaveOccurred()))

		filter.Spec.Priority = 2
		client := createFakeClient(&filter)

		sample := NewEnvoyFilterAllowPartialReferer(client)

		//when
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultUpdated))

		var s networkingv1alpha3.EnvoyFilterList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[EnvoyFilterAnnotation]).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[istio.DisclaimerKey]).To(Not(BeNil()))
	})
})

var _ = Describe("RequiresProxyRestart", func() {
	templateValues := map[string]string{}
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	It("should return true when pod was created before EnvoyFilter updated", func() {
		//given
		pod := createPod("test", "test", "Deployment", "owner")
		pod2 := createPod("test2", "test", "Deployment", "owner")
		t, err := time.Parse(time.RFC3339, "2006-01-02T15:04:05Z")
		Expect(err).To(Not(HaveOccurred()))
		pod.CreationTimestamp = metav1.Time{Time: t}
		pod2.CreationTimestamp = metav1.Time{Time: t}

		client := createFakeClient(pod, pod2)

		sample := NewEnvoyFilterAllowPartialReferer(client)
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		//when
		evaluator, err := sample.NewProxyRestartEvaluator(context.TODO())
		restart := evaluator.RequiresProxyRestart(*pod)
		restart2 := evaluator.RequiresProxyRestart(*pod2)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(restart).To(BeTrue())
		Expect(restart2).To(BeTrue())
	})

	It("should return false when pod was created after EnvoyFilter updated", func() {
		//given
		pod := createPod("test", "test", "Deployment", "owner")
		t, err := time.Parse(time.RFC3339, "2077-01-02T15:04:05Z")
		Expect(err).To(Not(HaveOccurred()))
		pod.CreationTimestamp = metav1.Time{Time: t}

		client := createFakeClient(pod)

		sample := NewEnvoyFilterAllowPartialReferer(client)
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		//when
		evaluator, err := sample.NewProxyRestartEvaluator(context.TODO())
		Expect(err).To(Not(HaveOccurred()))

		//then
		restart := evaluator.RequiresProxyRestart(*pod)
		Expect(restart).To(BeFalse())
	})
})

var _ = Describe("RequiresProxyRestart", func() {
	templateValues := map[string]string{}
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	It("should return false when pod was created after EnvoyFilter updated", func() {
		//given
		pod := createPod("test", "test", "Deployment", "owner")
		t, err := time.Parse(time.RFC3339, "2077-01-02T15:04:05Z")
		Expect(err).To(Not(HaveOccurred()))
		pod.CreationTimestamp = metav1.Time{Time: t}

		client := createFakeClient(pod)

		sample := NewEnvoyFilterAllowPartialReferer(client)
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		//when
		evaluator, err := sample.NewIngressGatewayEvaluator(context.TODO())
		Expect(err).To(Not(HaveOccurred()))

		//then
		restart := evaluator.RequiresIngressGatewayRestart(*pod)
		Expect(restart).To(BeFalse())
	})

	It("should return true when pod was created before EnvoyFilter updated", func() {
		//given
		pod := createPod("test", "test", "Deployment", "owner")
		t, err := time.Parse(time.RFC3339, "2000-01-02T15:04:05Z")
		Expect(err).To(Not(HaveOccurred()))
		pod.CreationTimestamp = metav1.Time{Time: t}

		client := createFakeClient(pod)

		sample := NewEnvoyFilterAllowPartialReferer(client)
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		//when
		evaluator, err := sample.NewIngressGatewayEvaluator(context.TODO())
		Expect(err).To(Not(HaveOccurred()))

		//then
		restart := evaluator.RequiresIngressGatewayRestart(*pod)
		Expect(restart).To(BeTrue())
	})
})

func createPod(name, namespace, containerName, imageVersion string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Annotations: map[string]string{
				"sidecar.istio.io/status": "ready",
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
					Name:  containerName,
					Image: "image:" + imageVersion,
				},
			},
		},
	}
}
