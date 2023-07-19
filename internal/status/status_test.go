package status_test

import (
	"context"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"
	"github.com/pkg/errors"
	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types2 "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func createFakeClient(objects ...client.Object) client.Client {
	err := operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1alpha3.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}

func TestStatus(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Istio Resources Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("istio-resources-suite", report)
})

var _ = Describe("SetReady", func() {
	It("Should update Istio CR status to ready", func() {
		// given
		handler := status.NewDefaultStatusHandler()

		cr := operatorv1alpha1.Istio{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		}
		k8sClient := createFakeClient(&cr)

		// when
		err := handler.UpdateToReady(context.TODO(), k8sClient, &cr)

		// then
		Expect(err).ToNot(HaveOccurred())

		err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)
		Expect(err).ToNot(HaveOccurred())
		Expect(cr.Status.State).To(Equal(operatorv1alpha1.Ready))
	})

	It("Should reset existing status description to empty", func() {
		// given
		handler := status.NewDefaultStatusHandler()

		cr := operatorv1alpha1.Istio{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			Status: operatorv1alpha1.IstioStatus{
				State:       operatorv1alpha1.Deleting,
				Description: "some description",
			},
		}

		k8sClient := createFakeClient(&cr)

		// when
		err := handler.UpdateToReady(context.TODO(), k8sClient, &cr)

		// then
		Expect(err).ToNot(HaveOccurred())

		Expect(k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)).Should(Succeed())
		Expect(cr.Status.State).To(Equal(operatorv1alpha1.Ready))
		Expect(cr.Status.Description).To(BeEmpty())
	})
})

var _ = Describe("UpdateToDeleting", func() {
	It("Should update Istio CR status to deleting", func() {
		// given
		handler := status.NewDefaultStatusHandler()

		cr := operatorv1alpha1.Istio{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		}
		k8sClient := createFakeClient(&cr)

		// when
		err := handler.UpdateToDeleting(context.TODO(), k8sClient, &cr)

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

var _ = Describe("UpdateToProcessing", func() {
	It("Should update Istio CR status to processing with description", func() {
		// given
		handler := status.NewDefaultStatusHandler()

		cr := operatorv1alpha1.Istio{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		}
		k8sClient := createFakeClient(&cr)

		// when
		err := handler.UpdateToProcessing(context.TODO(), "processing some stuff", k8sClient, &cr)

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

var _ = Describe("UpdateToError", func() {
	It("Should update Istio CR status to error with description", func() {
		// given
		handler := status.NewDefaultStatusHandler()

		cr := operatorv1alpha1.Istio{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		}
		k8sClient := createFakeClient(&cr)

		describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something")

		// when
		err := handler.UpdateToError(context.TODO(), describedError, k8sClient, &cr)

		// then
		Expect(err).ToNot(HaveOccurred())

		err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)
		Expect(err).ToNot(HaveOccurred())
		Expect(cr.Status.State).To(Equal(operatorv1alpha1.Error))
		Expect(cr.Status.Description).To(Equal("Something: error happened"))
	})

	It("Should update Istio CR status to warning with description", func() {
		// given
		handler := status.NewDefaultStatusHandler()

		cr := operatorv1alpha1.Istio{
			ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
		}
		k8sClient := createFakeClient(&cr)

		describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something").SetWarning()

		// when
		err := handler.UpdateToError(context.TODO(), describedError, k8sClient, &cr)

		// then
		Expect(err).ToNot(HaveOccurred())

		err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)
		Expect(err).ToNot(HaveOccurred())
		Expect(cr.Status.State).To(Equal(operatorv1alpha1.Warning))
		Expect(cr.Status.Description).To(Equal("Something: error happened"))
	})
})
