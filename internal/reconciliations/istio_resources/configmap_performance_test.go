package istio_resources

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/resources"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
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

	It("should return created if no resource was present", func() {
		//given
		client := resources.createFakeClient()
		sample := NewConfigMapPerformance(client)

		//when
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultCreated))

		var s corev1.ConfigMapList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[istio.DisclaimerKey]).To(Not(BeNil()))
	})

	It("should return updated if reapplied", func() {
		//given
		var p corev1.ConfigMap
		err := yaml.Unmarshal(manifest_cm_performance, &p)
		Expect(err).To(Not(HaveOccurred()))

		client := resources.createFakeClient(&p)

		sample := NewConfigMapPerformance(client)

		//when
		changed, err := sample.apply(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultUpdated))

		var s corev1.ConfigMapList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(1))

		Expect(s.Items[0].Annotations).To(Not(BeNil()))
		Expect(s.Items[0].Annotations[istio.DisclaimerKey]).To(Not(BeNil()))
	})
})
