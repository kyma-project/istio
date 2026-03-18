package istioresources

import (
	"context"

	"github.com/kyma-project/istio/operator/internal/resources"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("VPA", func() {
	templateValues := map[string]string{}
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	Context("when VPA CRD is present", func() {
		It("should create the VPA if no resource was present", func() {
			// given
			k8sClient := createFakeClient(createVPACRD())
			sample := NewVPA(false)

			// when
			changed, err := sample.reconcile(context.Background(), k8sClient, owner, templateValues)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultCreated))

			vpaList := unstructured.UnstructuredList{}
			vpaList.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "autoscaling.k8s.io",
				Version: "v1",
				Kind:    "VerticalPodAutoscalerList",
			})
			listErr := k8sClient.List(context.Background(), &vpaList)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(vpaList.Items).To(HaveLen(1))

			vpa := vpaList.Items[0]
			Expect(vpa.GetAnnotations()).To(Not(BeNil()))
			Expect(vpa.GetAnnotations()[resources.DisclaimerKey]).To(Not(BeEmpty()))

			Expect(vpa.GetLabels()).ToNot(BeNil())
			Expect(vpa.GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/version", "dev"))

			Expect(vpa.GetOwnerReferences()).To(HaveLen(1))
			Expect(vpa.GetOwnerReferences()[0].APIVersion).To(Equal(owner.APIVersion))
			Expect(vpa.GetOwnerReferences()[0].Kind).To(Equal(owner.Kind))
			Expect(vpa.GetOwnerReferences()[0].Name).To(Equal(owner.Name))
			Expect(vpa.GetOwnerReferences()[0].UID).To(Equal(owner.UID))
		})

		It("should return none if no change was applied", func() {
			// given
			k8sClient := createFakeClient(createVPACRD())
			sample := NewVPA(false)

			// first reconciliation
			changed, err := sample.reconcile(context.Background(), k8sClient, owner, templateValues)
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultCreated))

			// when - second reconciliation
			sample = NewVPA(false)
			changed, err = sample.reconcile(context.Background(), k8sClient, owner, templateValues)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultNone))
		})

		It("should delete the VPA when shouldDelete is true", func() {
			// given
			k8sClient := createFakeClient(createVPACRD())
			// first create the VPA
			createSample := NewVPA(false)
			_, err := createSample.reconcile(context.Background(), k8sClient, owner, templateValues)
			Expect(err).To(Not(HaveOccurred()))

			// when
			deleteSample := NewVPA(true)
			changed, err := deleteSample.reconcile(context.Background(), k8sClient, owner, templateValues)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultUpdated))

			vpaList := unstructured.UnstructuredList{}
			vpaList.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "autoscaling.k8s.io",
				Version: "v1",
				Kind:    "VerticalPodAutoscalerList",
			})
			listErr := k8sClient.List(context.Background(), &vpaList)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(vpaList.Items).To(BeEmpty())
		})
	})

	Context("when VPA CRD is not present", func() {
		It("should not create the VPA", func() {
			// given
			k8sClient := createFakeClient()
			sample := NewVPA(false)

			// when
			changed, err := sample.reconcile(context.Background(), k8sClient, owner, templateValues)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultNone))

			vpaList := unstructured.UnstructuredList{}
			vpaList.SetGroupVersionKind(schema.GroupVersionKind{
				Group:   "autoscaling.k8s.io",
				Version: "v1",
				Kind:    "VerticalPodAutoscalerList",
			})
			listErr := k8sClient.List(context.Background(), &vpaList)
			Expect(listErr).To(Not(HaveOccurred()))
			Expect(vpaList.Items).To(BeEmpty())
		})

		It("should not delete the VPA when shouldDelete is true", func() {
			// given
			k8sClient := createFakeClient()
			sample := NewVPA(true)

			// when
			changed, err := sample.reconcile(context.Background(), k8sClient, owner, templateValues)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultNone))
		})
	})
})

func createVPACRD() *unstructured.Unstructured {
	crd := &unstructured.Unstructured{}
	crd.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "apiextensions.k8s.io",
		Version: "v1",
		Kind:    "CustomResourceDefinition",
	})
	crd.SetName(vpaCRDName)
	return crd
}
