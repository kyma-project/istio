package controllers

import (
	"context"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types2 "k8s.io/apimachinery/pkg/types"
)

var _ = Describe("status", func() {
	Describe("updateToReady", func() {

		It("Should update Istio CR status to ready", func() {
			// given
			cr := operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := newStatusHandler(k8sClient)

			// when
			err := handler.updateToReady(context.TODO(), &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha1.Ready))
		})

		It("Should reset existing status description to empty", func() {
			// given
			cr := operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Status: operatorv1alpha1.IstioStatus{
					State:       operatorv1alpha1.Deleting,
					Description: "some description",
				},
			}

			k8sClient := createFakeClient(&cr)
			handler := newStatusHandler(k8sClient)

			// when
			err := handler.updateToReady(context.TODO(), &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			Expect(k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)).Should(Succeed())
			Expect(cr.Status.State).To(Equal(operatorv1alpha1.Ready))
			Expect(cr.Status.Description).To(BeEmpty())
		})
	})

	Describe("updateToDeleting", func() {
		It("Should update Istio CR status to deleting", func() {
			// given
			cr := operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := newStatusHandler(k8sClient)

			// when
			err := handler.updateToDeleting(context.TODO(), &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			Expect(cr.Status.State).To(Equal(operatorv1alpha1.Deleting))
			Expect(cr.Status.Description).ToNot(BeEmpty())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha1.Deleting))
			Expect(cr.Status.Description).ToNot(BeEmpty())
		})
	})

	Describe("updateToProcessing", func() {
		It("Should update Istio CR status to processing with description", func() {
			// given
			cr := operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := newStatusHandler(k8sClient)

			// when
			err := handler.updateToProcessing(context.TODO(), "processing some stuff", &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			Expect(cr.Status.State).To(Equal(operatorv1alpha1.Processing))
			Expect(cr.Status.Description).To(Equal("processing some stuff"))

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha1.Processing))
			Expect(cr.Status.Description).To(Equal("processing some stuff"))
		})
	})

	Describe("updateToError", func() {
		It("Should update Istio CR status to error with description", func() {
			// given
			cr := operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := newStatusHandler(k8sClient)

			describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something")

			// when
			err := handler.updateToError(context.TODO(), describedError, "", &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha1.Error))
			Expect(cr.Status.Description).To(Equal("Something: error happened"))
		})

		It("Should update Istio CR status to warning with description", func() {
			// given
			cr := operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := newStatusHandler(k8sClient)

			describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something").SetWarning()

			// when
			err := handler.updateToError(context.TODO(), describedError, "", &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)
			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha1.Warning))
			Expect(cr.Status.Description).To(Equal("Something: error happened"))
		})
	})
})
