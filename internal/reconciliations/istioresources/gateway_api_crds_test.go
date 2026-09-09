//go:build !experimental

package istioresources

import (
	"context"

	"github.com/kyma-project/istio/operator/pkg/labels"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

var _ = Describe("GatewayAPICRDs", func() {
	owner := metav1.OwnerReference{
		APIVersion: "operator.kyma-project.io/v1alpha2",
		Kind:       "Istio",
		Name:       "owner-name",
		UID:        "owner-uid",
	}
	templateValues := map[string]string{}

	Context("install path (shouldDelete=false)", func() {
		It("should create CRDs with module label when none are present", func() {
			fakeClient := createFakeClient()
			r := NewGatewayAPICRDs(false)

			result, err := r.reconcile(context.Background(), fakeClient, owner, templateValues)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(controllerutil.OperationResultUpdated))

			// All 7 CRDs should be created
			for _, manifest := range gatewayAPICRDManifests {
				var desired unstructured.Unstructured
				Expect(yaml.Unmarshal(manifest, &desired)).To(Succeed())

				existing := unstructured.Unstructured{}
				existing.SetGroupVersionKind(desired.GroupVersionKind())
				Expect(fakeClient.Get(context.Background(), client.ObjectKey{Name: desired.GetName()}, &existing)).To(Succeed())
				Expect(existing.GetLabels()).To(HaveKeyWithValue(labels.ModuleLabelKey, labels.ModuleLabelValue))
			}
		})

		It("should update module-managed CRDs that already exist", func() {
			// Pre-create one CRD with module label
			var desired unstructured.Unstructured
			Expect(yaml.Unmarshal(gatewayAPIGatewaysCRD, &desired)).To(Succeed())
			lbls := map[string]string{labels.ModuleLabelKey: labels.ModuleLabelValue}
			desired.SetLabels(lbls)
			fakeClient := createFakeClient(&desired)

			r := NewGatewayAPICRDs(false)
			_, err := r.reconcile(context.Background(), fakeClient, owner, templateValues)

			Expect(err).NotTo(HaveOccurred())

			var updated unstructured.Unstructured
			updated.SetGroupVersionKind(desired.GroupVersionKind())
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Name: desired.GetName()}, &updated)).To(Succeed())
			Expect(updated.GetLabels()).To(HaveKeyWithValue(labels.ModuleLabelKey, labels.ModuleLabelValue))
		})

		It("should preserve existing labels and annotations on module-managed CRDs across reconciliations", func() {
			// First reconciliation — CRDs are created and the disclaimer is annotated.
			fakeClient := createFakeClient()
			r := NewGatewayAPICRDs(false)

			_, err := r.reconcile(context.Background(), fakeClient, owner, templateValues)
			Expect(err).NotTo(HaveOccurred())

			// Simulate AnnotateWithDisclaimer adding the annotation (done by the caller in production).
			var crd unstructured.Unstructured
			Expect(yaml.Unmarshal(gatewayAPIGatewaysCRD, &crd)).To(Succeed())
			existing := unstructured.Unstructured{}
			existing.SetGroupVersionKind(crd.GroupVersionKind())
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Name: crd.GetName()}, &existing)).To(Succeed())

			ann := existing.GetAnnotations()
			if ann == nil {
				ann = make(map[string]string)
			}
			ann["istios.operator.kyma-project.io/managed-by-disclaimer"] = "DO NOT EDIT"
			ann["user-annotation"] = "preserved"
			existing.SetAnnotations(ann)
			existingLabels := existing.GetLabels()
			existingLabels["user-label"] = "preserved"
			existing.SetLabels(existingLabels)
			Expect(fakeClient.Update(context.Background(), &existing)).To(Succeed())

			// Second reconciliation — should not strip the disclaimer or user metadata.
			_, err = r.reconcile(context.Background(), fakeClient, owner, templateValues)
			Expect(err).NotTo(HaveOccurred())

			var updated unstructured.Unstructured
			updated.SetGroupVersionKind(crd.GroupVersionKind())
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Name: crd.GetName()}, &updated)).To(Succeed())

			Expect(updated.GetAnnotations()).To(HaveKeyWithValue("istios.operator.kyma-project.io/managed-by-disclaimer", "DO NOT EDIT"))
			Expect(updated.GetAnnotations()).To(HaveKeyWithValue("user-annotation", "preserved"))
			Expect(updated.GetLabels()).To(HaveKeyWithValue("user-label", "preserved"))
			Expect(updated.GetLabels()).To(HaveKeyWithValue(labels.ModuleLabelKey, labels.ModuleLabelValue))
		})

		It("should return a warning and not modify CRDs that exist without module label", func() {
			// Pre-create a CRD without module label
			var desired unstructured.Unstructured
			Expect(yaml.Unmarshal(gatewayAPIHTTPRoutesCRD, &desired)).To(Succeed())
			desired.SetLabels(map[string]string{"some-other-label": "value"})
			fakeClient := createFakeClient(&desired)

			r := NewGatewayAPICRDs(false)
			_, err := r.reconcile(context.Background(), fakeClient, owner, templateValues)

			Expect(err).To(HaveOccurred())
			warn, ok := err.(*unmanagedCRDsWarning)
			Expect(ok).To(BeTrue())
			Expect(warn.names).To(ContainElement(desired.GetName()))

			// CRD should not have been modified — original label still present, no module label
			var existing unstructured.Unstructured
			existing.SetGroupVersionKind(desired.GroupVersionKind())
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Name: desired.GetName()}, &existing)).To(Succeed())
			Expect(existing.GetLabels()).NotTo(HaveKey(labels.ModuleLabelKey))
		})
	})

	Context("delete path (shouldDelete=true)", func() {
		It("should delete module-managed CRDs", func() {
			var desired unstructured.Unstructured
			Expect(yaml.Unmarshal(gatewayAPIGatewayClassesCRD, &desired)).To(Succeed())
			desired.SetLabels(map[string]string{labels.ModuleLabelKey: labels.ModuleLabelValue})
			fakeClient := createFakeClient(&desired)

			r := NewGatewayAPICRDs(true)
			_, err := r.reconcile(context.Background(), fakeClient, owner, templateValues)

			Expect(err).NotTo(HaveOccurred())

			var remaining unstructured.Unstructured
			remaining.SetGroupVersionKind(desired.GroupVersionKind())
			getErr := fakeClient.Get(context.Background(), client.ObjectKey{Name: desired.GetName()}, &remaining)
			Expect(getErr).To(HaveOccurred())
		})

		It("should not delete CRDs that exist without module label", func() {
			var desired unstructured.Unstructured
			Expect(yaml.Unmarshal(gatewayAPIGatewayClassesCRD, &desired)).To(Succeed())
			desired.SetLabels(map[string]string{"user-label": "user-value"})
			fakeClient := createFakeClient(&desired)

			r := NewGatewayAPICRDs(true)
			_, err := r.reconcile(context.Background(), fakeClient, owner, templateValues)

			Expect(err).NotTo(HaveOccurred())

			var remaining unstructured.Unstructured
			remaining.SetGroupVersionKind(desired.GroupVersionKind())
			Expect(fakeClient.Get(context.Background(), client.ObjectKey{Name: desired.GetName()}, &remaining)).To(Succeed())
		})

		It("should succeed when no CRDs are present", func() {
			fakeClient := createFakeClient()

			r := NewGatewayAPICRDs(true)
			result, err := r.reconcile(context.Background(), fakeClient, owner, templateValues)

			Expect(err).NotTo(HaveOccurred())
			Expect(result).To(Equal(controllerutil.OperationResultUpdated))
		})
	})
})
