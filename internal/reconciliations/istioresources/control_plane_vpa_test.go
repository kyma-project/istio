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

var _ = Describe("ControlPlaneVPA", func() {
	templateValues := map[string]string{}
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}

	Context("when VPA CRD is present", func() {
		It("should create all control plane VPAs", func() {
			// given
			k8sClient := createFakeClient(createVPACRD())
			sample := NewControlPlaneVPA(false)

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
			Expect(vpaList.Items).To(HaveLen(4))

			expectedNames := []string{"istiod-vpa", "istio-ingressgateway-vpa", "istio-egressgateway-vpa", "istio-cni-node-vpa"}
			for _, vpa := range vpaList.Items {
				Expect(expectedNames).To(ContainElement(vpa.GetName()))
				Expect(vpa.GetNamespace()).To(Equal("istio-system"))
				Expect(vpa.GetAnnotations()).To(Not(BeNil()))
				Expect(vpa.GetAnnotations()[resources.DisclaimerKey]).To(Not(BeEmpty()))
				Expect(vpa.GetLabels()).ToNot(BeNil())
				Expect(vpa.GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/version", "dev"))
				Expect(vpa.GetOwnerReferences()).To(BeEmpty())
			}
		})

		It("should return none if no change was applied on second reconciliation", func() {
			// given
			k8sClient := createFakeClient(createVPACRD())
			sample := NewControlPlaneVPA(false)

			// first reconciliation
			changed, err := sample.reconcile(context.Background(), k8sClient, owner, templateValues)
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultCreated))

			// when - second reconciliation
			sample = NewControlPlaneVPA(false)
			changed, err = sample.reconcile(context.Background(), k8sClient, owner, templateValues)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultNone))
		})

		It("should delete all control plane VPAs when shouldDelete is true", func() {
			// given
			k8sClient := createFakeClient(createVPACRD())
			createSample := NewControlPlaneVPA(false)
			_, err := createSample.reconcile(context.Background(), k8sClient, owner, templateValues)
			Expect(err).To(Not(HaveOccurred()))

			// when
			deleteSample := NewControlPlaneVPA(true)
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
		It("should not create any control plane VPAs", func() {
			// given
			k8sClient := createFakeClient()
			sample := NewControlPlaneVPA(false)

			// when
			changed, err := sample.reconcile(context.Background(), k8sClient, owner, templateValues)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultNone))
		})

		It("should not attempt delete when VPA CRD is not present", func() {
			// given
			k8sClient := createFakeClient()
			sample := NewControlPlaneVPA(true)

			// when
			changed, err := sample.reconcile(context.Background(), k8sClient, owner, templateValues)

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(changed).To(Equal(controllerutil.OperationResultNone))
		})
	})
})
