package controllers

import (
	"context"
	"github.com/go-logr/logr"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	istioCrName                = "default"
	testNamespace              = "kyma-system"
	testReconciliationInterval = time.Second * 5
)

var _ = Describe("Istio Controller", func() {
	Context("Reconcile", func() {
		It("should fail to reconcile Istio CR in different than kyma-system namespace and set error state", func() {
			//given
			numTrustedProxies := 1
			istioCR := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
				Name:            "default",
				ResourceVersion: "1",
			},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: &numTrustedProxies,
					},
				},
			}

			client := createFakeClient(&istioCR)
			istioController := &IstioReconciler{
				Client:                 client,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          newStatusHandler(client),
				reconciliationInterval: 10 * time.Hour,
			}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "default",
				},
			}

			//when
			res, err := istioController.Reconcile(context.TODO(), req)

			//then
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("istio CR is not in kyma-system namespace"))
			Expect(res.Requeue).To(BeFalse())

			processedIstioCR := operatorv1alpha1.Istio{}
			err = client.Get(context.TODO(), types.NamespacedName{Name: "default"}, &processedIstioCR)
			Expect(err).To(Not(HaveOccurred()))
			Expect(processedIstioCR.Status.State).To(Equal(operatorv1alpha1.Error))
			Expect(processedIstioCR.Status.Description).To(Equal("Stopped Istio CR reconciliation: istio CR is not in kyma-system namespace"))
		})

		It("Should not return an error when CR was not found", func() {
			// given
			apiClient := createFakeClient()

			sut := &IstioReconciler{
				Client:                 apiClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          &StatusMock{},
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{}))
		})

		It("Should call update status to processing when CR is not deleted", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
				},
			}

			statusMock := StatusMock{}
			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          &statusMock,
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{}))
			Expect(statusMock.updatedToProcessingCalled).Should(BeTrue())
		})

		It("Should return an error when update status to processing failed", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
				},
			}

			statusMock := StatusMock{
				processingError: errors.New("Update to processing error"),
			}
			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          &statusMock,
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err.Error()).To(Equal("Update to processing error"))
			Expect(result).Should(Equal(reconcile.Result{}))
			Expect(statusMock.updatedToProcessingCalled).Should(BeTrue())
		})

		It("Should call update status to deleting when CR is deleted", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
				},
			}
			statusMock := StatusMock{}
			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          &statusMock,
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{}))
			Expect(statusMock.updatedToDeletingCalled).Should(BeTrue())
		})

		It("Should return an error when update status to deleting failed", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
				},
			}

			statusMock := StatusMock{
				deletingError: errors.New("Update to deleting error"),
			}
			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          &statusMock,
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err.Error()).To(Equal("Update to deleting error"))
			Expect(result).Should(Equal(reconcile.Result{}))
			Expect(statusMock.updatedToDeletingCalled).Should(BeTrue())
		})

		It("Should not requeue a deleted CR when there are no finalizers", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
				},
			}

			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          newStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{}))

			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), istioCR)
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("Should set ready status, update lastAppliedConfiguration annotation and requeue when successfully reconciled", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: pointer.Int(2),
					},
				},
			}

			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          newStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{RequeueAfter: testReconciliationInterval}))

			Expect(fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), istioCR)).Should(Succeed())
			Expect(istioCR.Status.State).Should(Equal(operatorv1alpha1.Ready))
			Expect(istioCR.Annotations["operator.kyma-project.io/lastAppliedConfiguration"]).To(ContainSubstring("{\"config\":{\"numTrustedProxies\":2},"))
		})

		It("Should return an error when update status to ready failed", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
			}
			statusMock := StatusMock{
				readyError: errors.New("Update to ready error"),
			}
			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          &statusMock,
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err.Error()).To(Equal("Update to ready error"))
			Expect(result).Should(Equal(reconcile.Result{}))
			Expect(statusMock.updatedToReadyCalled).Should(BeTrue())
		})

		It("Should set error status and return an error when Istio installation reconciliation failed", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
			}

			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client: fakeClient,
				Scheme: getTestScheme(),
				istioInstallation: &istioInstallationReconciliationMock{
					err: described_errors.NewDescribedError(errors.New("istio test error"), "test error description"),
				},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          newStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("istio test error"))
			Expect(result).Should(Equal(reconcile.Result{}))

			Expect(fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), istioCR)).Should(Succeed())
			Expect(istioCR.Status.State).Should(Equal(operatorv1alpha1.Error))
			Expect(istioCR.Status.Description).To(ContainSubstring("test error description"))
		})

		It("Should set error status and return an error when proxy sidecar reconciliation failed", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
			}

			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:            fakeClient,
				Scheme:            getTestScheme(),
				istioInstallation: &istioInstallationReconciliationMock{},
				proxySidecars: &proxySidecarsReconciliationMock{
					err: errors.New("sidecar test error"),
				},
				log:                    logr.Discard(),
				statusHandler:          newStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("sidecar test error"))
			Expect(result).Should(Equal(reconcile.Result{}))

			Expect(fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), istioCR)).Should(Succeed())
			Expect(istioCR.Status.State).Should(Equal(operatorv1alpha1.Error))
			Expect(istioCR.Status.Description).To(ContainSubstring("Error occurred during reconciliation of Istio Sidecars"))
		})
	})
})

type istioInstallationReconciliationMock struct {
	err described_errors.DescribedError
}

func (i *istioInstallationReconciliationMock) Reconcile(_ context.Context, istioCR operatorv1alpha1.Istio, _ string) (operatorv1alpha1.Istio, described_errors.DescribedError) {
	return istioCR, i.err
}

type proxySidecarsReconciliationMock struct {
	err error
}

func (p *proxySidecarsReconciliationMock) Reconcile(_ context.Context, _ operatorv1alpha1.Istio) error {
	return p.err
}

type StatusMock struct {
	processingError           error
	updatedToProcessingCalled bool
	readyError                error
	updatedToReadyCalled      bool
	deletingError             error
	updatedToDeletingCalled   bool
	errorError                error
	updatedToErrorCalled      bool
}

func (s *StatusMock) updateToProcessing(_ context.Context, _ string, _ *operatorv1alpha1.Istio) error {
	s.updatedToProcessingCalled = true
	return s.processingError
}

func (s *StatusMock) updateToError(_ context.Context, _ described_errors.DescribedError, _ *operatorv1alpha1.Istio) error {
	s.updatedToErrorCalled = true
	return s.errorError
}

func (s *StatusMock) updateToDeleting(_ context.Context, _ *operatorv1alpha1.Istio) error {
	s.updatedToDeletingCalled = true
	return s.deletingError
}

func (s *StatusMock) updateToReady(_ context.Context, _ *operatorv1alpha1.Istio) error {
	s.updatedToReadyCalled = true
	return s.readyError
}
