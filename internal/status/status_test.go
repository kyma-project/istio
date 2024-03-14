package status

import (
	"context"
	"testing"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/tests"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
	securityv1beta1 "istio.io/client-go/pkg/apis/security/v1beta1"
	"k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	types2 "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestManifest(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Status Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("status-suite", report)
})

var _ = Describe("status", func() {
	Describe("UpdateToReady", func() {
		It("should update Istio CR status to ready", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			// when
			err := handler.UpdateToReady(context.TODO(), &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)

			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Ready))
			Expect(cr.Status.Conditions).To(BeNil())
		})

		It("should reset existing status description to empty", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
				Status: operatorv1alpha2.IstioStatus{
					State:       operatorv1alpha2.Deleting,
					Description: "some description",
				},
			}

			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			// when
			err := handler.UpdateToReady(context.TODO(), &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			Expect(k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)).Should(Succeed())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Ready))
			Expect(cr.Status.Description).To(BeEmpty())
			Expect(cr.Status.Conditions).To(BeNil())
		})
	})

	Describe("UpdateToDeleting", func() {
		It("should update Istio CR status to deleting", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			// when
			err := handler.UpdateToDeleting(context.TODO(), &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Deleting))
			Expect(cr.Status.Description).ToNot(BeEmpty())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)

			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Deleting))
			Expect(cr.Status.Description).ToNot(BeEmpty())
			Expect(cr.Status.Conditions).To(BeNil())
		})
	})

	Describe("UpdateToProcessing", func() {
		It("should update Istio CR status to processing with description", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			// when
			err := handler.UpdateToProcessing(context.TODO(), &cr)

			// then
			Expect(err).ToNot(HaveOccurred())

			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Processing))
			Expect(cr.Status.Description).To(Equal("Reconciling Istio"))

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)

			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Processing))
			Expect(cr.Status.Description).To(Equal("Reconciling Istio"))
			Expect(cr.Status.Conditions).To(BeNil())
		})
	})

	Describe("UpdateToError", func() {
		It("should update Istio CR status to error with description", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something")

			// when
			err := handler.UpdateToError(context.TODO(), &cr, describedError)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)

			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Error))
			Expect(cr.Status.Description).To(Equal("Something: error happened"))
			Expect(cr.Status.Conditions).To(BeNil())
		})

		It("should update Istio CR status to warning with description", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something").SetWarning()

			// when
			err := handler.UpdateToError(context.TODO(), &cr, describedError)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)

			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Warning))
			Expect(cr.Status.Description).To(Equal("Something: error happened"))
			Expect(cr.Status.Conditions).To(BeNil())
		})

		It("should update Istio CR status to error with default condition", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something")

			// when
			err := handler.UpdateToError(context.TODO(), &cr, describedError)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)

			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Error))
			Expect(cr.Status.Description).To(Equal("Something: error happened"))
			Expect(cr.Status.Conditions).To(BeNil())
		})

		It("should update Istio CR status to warning with a speicific condition derived from the error", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something").SetWarning()

			// when
			err := handler.UpdateToError(context.TODO(), &cr, describedError)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)

			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Warning))
			Expect(cr.Status.Description).To(Equal("Something: error happened"))
			Expect(cr.Status.Conditions).To(BeNil())
		})

		It("should update Istio CR status to warning and add ready condition if not provided", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			describedError := described_errors.NewDescribedError(errors.New("error happened"), "Something").SetWarning()

			// when
			err := handler.UpdateToError(context.TODO(), &cr, describedError)

			// then
			Expect(err).ToNot(HaveOccurred())

			err = k8sClient.Get(context.TODO(), types2.NamespacedName{Name: "test", Namespace: "default"}, &cr)

			Expect(err).ToNot(HaveOccurred())
			Expect(cr.Status.State).To(Equal(operatorv1alpha2.Warning))
			Expect(cr.Status.Description).To(Equal("Something: error happened"))
			Expect(cr.Status.Conditions).To(BeNil())
		})
	})

	Describe("SetCondition", func() {
		It("should set Istio CR status conditions", func() {
			// given
			cr := operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{Name: "test", Namespace: "default"},
			}
			k8sClient := createFakeClient(&cr)
			handler := NewStatusHandler(k8sClient)

			// when
			handler.SetCondition(&cr, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonReconcileSucceeded))
			handler.SetCondition(&cr, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonProxySidecarManualRestartRequired))

			// then
			Expect(cr.Status.Conditions).ToNot(BeNil())
			Expect((*cr.Status.Conditions)).To(HaveLen(2))
			Expect((*cr.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*cr.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonReconcileSucceeded)))
			Expect((*cr.Status.Conditions)[0].Status).To(Equal(metav1.ConditionTrue))

			Expect((*cr.Status.Conditions)[1].Type).To(Equal(string(operatorv1alpha2.ConditionTypeProxySidecarRestartSucceeded)))
			Expect((*cr.Status.Conditions)[1].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonProxySidecarManualRestartRequired)))
			Expect((*cr.Status.Conditions)[1].Status).To(Equal(metav1.ConditionFalse))
		})
	})
})

func createFakeClient(objects ...client.Object) client.Client {
	return fake.NewClientBuilder().WithScheme(getTestScheme()).WithObjects(objects...).WithStatusSubresource(objects...).Build()
}

func getTestScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	Expect(operatorv1alpha2.AddToScheme(scheme)).Should(Succeed())
	Expect(v1alpha3.AddToScheme(scheme)).Should(Succeed())
	Expect(v1beta1.AddToScheme(scheme)).Should(Succeed())
	Expect(securityv1beta1.AddToScheme(scheme)).Should(Succeed())

	return scheme
}
