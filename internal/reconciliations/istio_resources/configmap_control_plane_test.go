package istio_resources

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("Apply", func() {
	templateValues := map[string]string{}
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	It("should return unchanged if no resource was present", func() {
		//given
		client := createFakeClient()
		sample := NewConfigMapControlPlane(client)

		//when
		changed, err := sample.reconcile(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(Equal(controllerutil.OperationResultNone))

		var s corev1.ConfigMapList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(0))
	})

	It("should return deleted if present", func() {
		//given
		p := corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      controlPlaneDashboardName,
				Namespace: controlPlaneDashboardNamespace,
			},
		}

		client := createFakeClient(&p)

		sample := NewConfigMapControlPlane(client)

		//when
		changed, err := sample.reconcile(context.TODO(), client, owner, templateValues)

		//then
		Expect(err).To(Not(HaveOccurred()))
		Expect(changed).To(BeEquivalentTo("deleted"))

		var s corev1.ConfigMapList
		listErr := client.List(context.TODO(), &s)
		Expect(listErr).To(Not(HaveOccurred()))
		Expect(s.Items).To(HaveLen(0))
	})
})
