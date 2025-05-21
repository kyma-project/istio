package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/restarter"
	"k8s.io/utils/ptr"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	"github.com/kyma-project/istio/operator/internal/status"

	"github.com/go-logr/logr"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
			istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
				Name:            "default",
				ResourceVersion: "1",
			},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						NumTrustedProxies: &numTrustedProxies,
					},
				},
			}

			client := createFakeClient(&istioCR)
			istioController := &IstioReconciler{
				Client:                 client,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(client),
				reconciliationInterval: 10 * time.Hour,
			}
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name: "default",
				},
			}

			//when
			result, err := istioController.Reconcile(context.Background(), req)

			//then
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			updatedIstioCR := operatorv1alpha2.Istio{}
			err = client.Get(context.Background(), types.NamespacedName{Name: "default"}, &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).To(Equal(operatorv1alpha2.Error))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Stopped Istio CR reconciliation: istio CR is not in kyma-system namespace"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonReconcileFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		})

		It("should not return an error when CR was not found", func() {
			// given
			apiClient := createFakeClient()

			sut := &IstioReconciler{
				Client:                 apiClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				log:                    logr.Discard(),
				statusHandler:          NewStatusMock(),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{}))
		})

		It("should call update status to processing when CR is not deleted", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
				},
			}

			statusMock := NewStatusMock()
			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          statusMock,
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{}))
			Expect(statusMock.updatedToProcessingCalled).Should(BeTrue())
		})

		It("should return an error when update status to processing failed", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
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
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
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

		It("should call update status to deleting when CR is deleted", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
					Finalizers: []string{"istios.operator.kyma-project.io/test-mock"},
				},
			}
			statusMock := NewStatusMock()
			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          statusMock,
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			_, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(statusMock.updatedToDeletingCalled).Should(BeTrue())
		})

		It("should return an error when update status to deleting failed", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					DeletionTimestamp: &metav1.Time{
						Time: time.Now(),
					},
					Finalizers: []string{"istios.operator.kyma-project.io/test-mock"},
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
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
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

		It("should not requeue a CR without finalizers, because it's considered to be in deletion", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
				},
			}

			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{}))
		})

		It("should set Ready status, update lastAppliedConfiguration annotation and requeue when successfully reconciled", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						NumTrustedProxies: ptr.To(2),
					},
				},
			}

			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{RequeueAfter: testReconciliationInterval}))

			updatedIstioCR := operatorv1alpha2.Istio{}
			err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Ready))
			Expect(updatedIstioCR.Annotations["operator.kyma-project.io/lastAppliedConfiguration"]).To(ContainSubstring("{\"config\":{\"numTrustedProxies\":2,\"telemetry\":{\"metrics\":{}}},"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonReconcileSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionTrue))
		})

		It("should return an error when update status to ready failed", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
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
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
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
			Expect(statusMock.setConditionCalled).Should(BeTrue())
			Expect(statusMock.GetConditions()).Should(Equal([]operatorv1alpha2.ReasonWithMessage{
				operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonReconcileUnknown),
				operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonReconcileSucceeded),
			}))
		})

		It("should set error status and return an error when Istio installation reconciliation failed", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
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
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("istio test error"))
			Expect(result).Should(Equal(reconcile.Result{}))

			updatedIstioCR := operatorv1alpha2.Istio{}
			err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Error))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("test error description"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioInstallUninstallFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		})

		It("should set error status and return an error when proxy sidecar reconciliation failed", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
			}

			fakeClient := createFakeClient(istioCR)

			sidecarsRestarter := &restarterMock{
				err: described_errors.NewDescribedError(errors.New("sidecar test error"), "Error occurred during reconciliation of Istio Sidecars"),
			}

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{sidecarsRestarter, &restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("sidecar test error"))
			Expect(result).Should(Equal(reconcile.Result{}))

			updatedIstioCR := operatorv1alpha2.Istio{}
			err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Error))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Error occurred during reconciliation of Istio Sidecars"))
		})

		It("should set ready status when successfully reconciled oldest Istio CR", func() {
			// given
			oldestIstioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:              istioCrName,
					Namespace:         testNamespace,
					UID:               "1",
					CreationTimestamp: metav1.Unix(1494505756, 0),
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						NumTrustedProxies: ptr.To(2),
					},
				},
			}
			newerIstioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:              fmt.Sprintf("%s-2", istioCrName),
					Namespace:         testNamespace,
					UID:               "2",
					CreationTimestamp: metav1.Now(),
				},
			}

			fakeClient := createFakeClient(oldestIstioCR, newerIstioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result.RequeueAfter).Should(Equal(testReconciliationInterval))

			updatedIstioCR := operatorv1alpha2.Istio{}
			err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(oldestIstioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Ready))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonReconcileSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionTrue))
		})

		It("should set an warning status and do not requeue an Istio CR when an older Istio CR is present", func() {
			// given
			oldestIstioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:              istioCrName,
					Namespace:         testNamespace,
					UID:               "1",
					CreationTimestamp: metav1.Unix(1494505756, 0),
				},
			}
			newerIstioCRName := fmt.Sprintf("%s-2", istioCrName)
			newerIstioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:              newerIstioCRName,
					Namespace:         testNamespace,
					UID:               "2",
					CreationTimestamp: metav1.Now(),
				},
			}

			fakeClient := createFakeClient(oldestIstioCR, newerIstioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: newerIstioCRName}})

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			updatedIstioCR := operatorv1alpha2.Istio{}
			err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(newerIstioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Warning))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring(fmt.Sprintf("only Istio CR %s in %s reconciles the module", istioCrName, testNamespace)))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonOlderCRExists)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		})

		It("should set an error status and requeue an Istio CR when is unable to list Istio CRs", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
				},
			}

			fakeClient := createFakeClient(istioCR)
			failClient := &shouldFailClient{fakeClient, true}
			sut := &IstioReconciler{
				Client:                 failClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			updatedIstioCR := operatorv1alpha2.Istio{}
			err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Error))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Unable to list Istio CRs"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonReconcileFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		})

		It("should set a warning if authorizer name is not unique", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:              istioCrName,
					Namespace:         testNamespace,
					UID:               "1",
					CreationTimestamp: metav1.Unix(1494505756, 0),
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
				Spec: operatorv1alpha2.IstioSpec{Config: operatorv1alpha2.Config{
					Authorizers: []*operatorv1alpha2.Authorizer{
						{
							Name:    "test-authorizer",
							Service: "test",
							Port:    2318,
						},
						{
							Name:    "test-authorizer",
							Service: "test2",
							Port:    2317,
						},
					},
				}},
			}

			fakeClient := createFakeClient(istioCR)
			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).To(Not(HaveOccurred()))
			Expect(result.Requeue).To(BeFalse())

			updatedIstioCR := operatorv1alpha2.Istio{}
			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Warning))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Authorizer name needs to be unique: test-authorizer is duplicated"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonValidationFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		})

		It("should update lastTransitionTime of Ready condition when reason changed", func() {
			// given
			istioCR := &operatorv1alpha2.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:      istioCrName,
					Namespace: testNamespace,
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
			}

			fakeClient := createFakeClient(istioCR)

			By("Mocking Istio install reconciliation to fail")
			reconcilerFailingOnIstioInstall := &IstioReconciler{
				Client: fakeClient,
				Scheme: getTestScheme(),
				istioInstallation: &istioInstallationReconciliationMock{
					err: described_errors.NewDescribedError(errors.New("test error"), "test error description"),
				},
				restarters:             []restarter.Restarter{&restarterMock{}},
				istioResources:         &istioResourcesReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			_, err := reconcilerFailingOnIstioInstall.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			Expect(err).To(HaveOccurred())

			updatedIstioCR := operatorv1alpha2.Istio{}
			Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)).Should(Succeed())

			By("Verifying that Istio CR has Condition Ready with False")
			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioInstallUninstallFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))

			firstNotReadyTransitionTime := (*updatedIstioCR.Status.Conditions)[0].LastTransitionTime

			By("Mocking Istio resources reconciliation to fail")
			reconcilerFailingOnIstioResources := &IstioReconciler{
				Client:            fakeClient,
				Scheme:            getTestScheme(),
				istioInstallation: &istioInstallationReconciliationMock{},
				restarters:        []restarter.Restarter{&restarterMock{}},
				istioResources: &istioResourcesReconciliationMock{
					err: described_errors.NewDescribedError(errors.New("test error"), "test error description"),
				},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			_, err = reconcilerFailingOnIstioResources.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).To(HaveOccurred())
			updatedIstioCR = operatorv1alpha2.Istio{}
			Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)).Should(Succeed())

			By("Verifying that the condition lastTransitionTime is also updated when only the reason has changed")
			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonCRsReconcileFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))

			secondNotReadyTransitionTime := (*updatedIstioCR.Status.Conditions)[0].LastTransitionTime
			Expect(secondNotReadyTransitionTime.Compare(firstNotReadyTransitionTime.Time) >= 0).To(BeTrue())
		})

		Context("Restarters", func() {
			It("should restart if reconciliations are successful", func() {
				//given
				istioCR := &operatorv1alpha2.Istio{
					ObjectMeta: metav1.ObjectMeta{
						Name:              istioCrName,
						Namespace:         testNamespace,
						UID:               "1",
						CreationTimestamp: metav1.Unix(1494505756, 0),
						Finalizers: []string{
							"istios.operator.kyma-project.io/istio-installation",
						},
					},
				}

				fakeClient := createFakeClient(istioCR)
				ingressGatewayRestarter := &restarterMock{restarted: false}
				proxySidecarsRestarter := &restarterMock{restarted: false}
				sut := &IstioReconciler{
					Client:                 fakeClient,
					Scheme:                 getTestScheme(),
					istioInstallation:      &istioInstallationReconciliationMock{},
					istioResources:         &istioResourcesReconciliationMock{},
					restarters:             []restarter.Restarter{proxySidecarsRestarter, ingressGatewayRestarter},
					log:                    logr.Discard(),
					statusHandler:          status.NewStatusHandler(fakeClient),
					reconciliationInterval: testReconciliationInterval,
				}

				//when
				_, _ = sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

				//then
				Expect(ingressGatewayRestarter.RestartCalled()).To(BeTrue())
				Expect(proxySidecarsRestarter.RestartCalled()).To(BeTrue())
			})

			It("should not restart when istio installation reconciliations failed", func() {
				//given
				istioCR := &operatorv1alpha2.Istio{
					ObjectMeta: metav1.ObjectMeta{
						Name:              istioCrName,
						Namespace:         testNamespace,
						UID:               "1",
						CreationTimestamp: metav1.Unix(1494505756, 0),
						Finalizers: []string{
							"istios.operator.kyma-project.io/istio-installation",
						},
					},
				}

				fakeClient := createFakeClient(istioCR)
				ingressGatewayRestarter := &restarterMock{restarted: false}
				sidecarsRestarter := &restarterMock{restarted: false}
				sut := &IstioReconciler{
					Client: fakeClient,
					Scheme: getTestScheme(),
					istioInstallation: &istioInstallationReconciliationMock{
						err: described_errors.NewDescribedError(errors.New("istio test error"), "test error description"),
					},
					istioResources:         &istioResourcesReconciliationMock{},
					log:                    logr.Discard(),
					statusHandler:          status.NewStatusHandler(fakeClient),
					reconciliationInterval: testReconciliationInterval,
				}

				//when
				_, _ = sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

				//then
				Expect(ingressGatewayRestarter.RestartCalled()).To(BeFalse())
				Expect(sidecarsRestarter.RestartCalled()).To(BeFalse())
			})

			It("should always invoke all of the restarters even if one failed", func() {
				//given
				istioCR := &operatorv1alpha2.Istio{
					ObjectMeta: metav1.ObjectMeta{
						Name:              istioCrName,
						Namespace:         testNamespace,
						UID:               "1",
						CreationTimestamp: metav1.Unix(1494505756, 0),
						Finalizers: []string{
							"istios.operator.kyma-project.io/istio-installation",
						},
					},
				}

				fakeClient := createFakeClient(istioCR)
				proxySidecarsRestarter := &restarterMock{restarted: false,
					err: described_errors.NewDescribedError(errors.New("error during restart"), "error during restart"),
				}
				ingressGatewayRestarter := &restarterMock{restarted: false,
					err: described_errors.NewDescribedError(errors.New("also error during restart"), "also error during restart")}
				sut := &IstioReconciler{
					Client:                 fakeClient,
					Scheme:                 getTestScheme(),
					istioInstallation:      &istioInstallationReconciliationMock{},
					istioResources:         &istioResourcesReconciliationMock{},
					restarters:             []restarter.Restarter{proxySidecarsRestarter, ingressGatewayRestarter},
					log:                    logr.Discard(),
					statusHandler:          status.NewStatusHandler(fakeClient),
					reconciliationInterval: testReconciliationInterval,
				}

				//when
				_, _ = sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

				//then
				Expect(ingressGatewayRestarter.RestartCalled()).To(BeTrue())
				Expect(proxySidecarsRestarter.RestartCalled()).To(BeTrue())
			})

			It("should return error when one of restarters return error and other warning", func() {
				//given
				istioCR := &operatorv1alpha2.Istio{
					ObjectMeta: metav1.ObjectMeta{
						Name:              istioCrName,
						Namespace:         testNamespace,
						UID:               "1",
						CreationTimestamp: metav1.Unix(1494505756, 0),
						Finalizers: []string{
							"istios.operator.kyma-project.io/istio-installation",
						},
					},
				}

				fakeClient := createFakeClient(istioCR)
				ingressGatewayRestarter := &restarterMock{restarted: false, err: described_errors.NewDescribedError(errors.New("test described error"), "test error description")}
				proxySidecarsRestarter := &restarterMock{restarted: false, err: described_errors.NewDescribedError(errors.New("test described error"), "test error description").SetWarning()}
				sut := &IstioReconciler{
					Client:                 fakeClient,
					Scheme:                 getTestScheme(),
					istioInstallation:      &istioInstallationReconciliationMock{},
					istioResources:         &istioResourcesReconciliationMock{},
					restarters:             []restarter.Restarter{ingressGatewayRestarter, proxySidecarsRestarter},
					log:                    logr.Discard(),
					statusHandler:          status.NewStatusHandler(fakeClient),
					reconciliationInterval: testReconciliationInterval,
				}

				//when
				_, _ = sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

				//then
				updatedIstioCR := operatorv1alpha2.Istio{}
				Expect(fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)).Should(Succeed())

				Expect(updatedIstioCR.Status.State).To(Equal(operatorv1alpha2.Error))
			})

			It("should set status to warning if warning happened during restart", func() {
				// given
				istioCR := &operatorv1alpha2.Istio{
					ObjectMeta: metav1.ObjectMeta{
						Name:              istioCrName,
						Namespace:         testNamespace,
						UID:               "1",
						CreationTimestamp: metav1.Unix(1494505756, 0),
						Finalizers: []string{
							"istios.operator.kyma-project.io/istio-installation",
						},
					},
				}

				sidecarsRestarter := &restarterMock{
					err: described_errors.NewDescribedError(errors.New("Restart error"), "Restart Warning description").SetWarning(),
				}
				fakeClient := createFakeClient(istioCR)
				sut := &IstioReconciler{
					Client:                 fakeClient,
					Scheme:                 getTestScheme(),
					istioInstallation:      &istioInstallationReconciliationMock{},
					istioResources:         &istioResourcesReconciliationMock{},
					restarters:             []restarter.Restarter{sidecarsRestarter},
					log:                    logr.Discard(),
					statusHandler:          status.NewStatusHandler(fakeClient),
					reconciliationInterval: testReconciliationInterval,
				}

				// when
				result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

				// then
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Requeue).To(BeFalse())
				Expect(result.RequeueAfter).To(Equal(1 * time.Hour))

				updatedIstioCR := operatorv1alpha2.Istio{}
				err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
				Expect(err).To(Not(HaveOccurred()))

				Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha2.Warning))
				Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Restart Warning description"))
			})

			It("should requeue reconcile request when a restarter needs to finish work on the next reconcile", func() {
				//given
				istioCR := &operatorv1alpha2.Istio{
					ObjectMeta: metav1.ObjectMeta{
						Name:              istioCrName,
						Namespace:         testNamespace,
						UID:               "1",
						CreationTimestamp: metav1.Unix(1494505756, 0),
						Finalizers: []string{
							"istios.operator.kyma-project.io/istio-installation",
						},
					},
				}

				fakeClient := createFakeClient(istioCR)
				ingressGatewayRestarter := &restarterMock{restarted: false}
				proxySidecarsRestarter := &restarterMock{restarted: false, requeue: true}
				sut := &IstioReconciler{
					Client:                 fakeClient,
					Scheme:                 getTestScheme(),
					istioInstallation:      &istioInstallationReconciliationMock{},
					istioResources:         &istioResourcesReconciliationMock{},
					restarters:             []restarter.Restarter{ingressGatewayRestarter, proxySidecarsRestarter},
					log:                    logr.Discard(),
					statusHandler:          status.NewStatusHandler(fakeClient),
					reconciliationInterval: testReconciliationInterval,
				}

				//when
				reconcileResult, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

				//then
				Expect(err).ToNot(HaveOccurred())
				Expect(reconcileResult.Requeue).To(BeFalse())
				Expect(reconcileResult.RequeueAfter).To(Equal(time.Minute * 1))

				Expect(ingressGatewayRestarter.RestartCalled()).To(BeTrue())
				Expect(proxySidecarsRestarter.RestartCalled()).To(BeTrue())

				updatedIstioCR := operatorv1alpha2.Istio{}
				err = fakeClient.Get(context.Background(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
				Expect(err).To(Not(HaveOccurred()))

				Expect(updatedIstioCR.Status.State).To(Equal(operatorv1alpha2.Processing))

				By("Verifying that Istio CR has Condition Ready status with Requeued reason")
				Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
				Expect(*updatedIstioCR.Status.Conditions).To(HaveLen(1))
				Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
				Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonReconcileRequeued)))
				Expect((*updatedIstioCR.Status.Conditions)[0].Message).To(Equal(operatorv1alpha2.ConditionReasonReconcileRequeuedMessage))
				Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
			})
		})
	})
})

type restarterMock struct {
	err       described_errors.DescribedError
	requeue   bool
	restarted bool
}

func (i *restarterMock) RestartCalled() bool {
	return i.restarted
}

func (i *restarterMock) Restart(_ context.Context, _ *operatorv1alpha2.Istio) (described_errors.DescribedError, bool) {
	i.restarted = true
	return i.err, i.requeue
}

type istioResourcesReconciliationMock struct {
	err described_errors.DescribedError
}

func (i *istioResourcesReconciliationMock) AddReconcileResource(_ istio_resources.Resource) istio_resources.ResourcesReconciliation {
	return i
}

func (i *istioResourcesReconciliationMock) Reconcile(_ context.Context, _ operatorv1alpha2.Istio) described_errors.DescribedError {
	return i.err
}

type shouldFailClient struct {
	client.Client
	FailOnList bool
}

func (p *shouldFailClient) List(ctx context.Context, list client.ObjectList, _ ...client.ListOption) error {
	if p.FailOnList {
		return errors.New("intentionally failing client on client.List")
	}
	return p.Client.List(ctx, list)
}

type istioInstallationReconciliationMock struct {
	err described_errors.DescribedError
}

func (i *istioInstallationReconciliationMock) Reconcile(_ context.Context, _ *operatorv1alpha2.Istio, _ status.Status) (istiooperator.IstioImageVersion, described_errors.DescribedError) {
	version, err := istiooperator.NewIstioImageVersionFromTag("1.16.0-distroless")
	if err != nil {
		i.err = described_errors.NewDescribedError(err, "error creating IstioImageVersion")
	}
	return version, i.err
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
	setConditionCalled        bool
	reasons                   []operatorv1alpha2.ReasonWithMessage
}

func NewStatusMock() *StatusMock {
	return &StatusMock{
		reasons: []operatorv1alpha2.ReasonWithMessage{},
	}
}

func (s *StatusMock) UpdateToProcessing(_ context.Context, _ *operatorv1alpha2.Istio) error {
	s.updatedToProcessingCalled = true
	return s.processingError
}

func (s *StatusMock) UpdateToDeleting(_ context.Context, _ *operatorv1alpha2.Istio) error {
	s.updatedToDeletingCalled = true
	return s.deletingError
}

func (s *StatusMock) UpdateToReady(_ context.Context, _ *operatorv1alpha2.Istio) error {
	s.updatedToReadyCalled = true
	return s.readyError
}

func (s *StatusMock) UpdateToError(_ context.Context, _ *operatorv1alpha2.Istio, _ described_errors.DescribedError, _ ...time.Duration) error {
	s.updatedToErrorCalled = true
	return s.errorError
}

func (s *StatusMock) SetCondition(_ *operatorv1alpha2.Istio, reason operatorv1alpha2.ReasonWithMessage) {
	s.setConditionCalled = true
	s.reasons = append(s.reasons, reason)
}

func (s *StatusMock) GetConditions() []operatorv1alpha2.ReasonWithMessage {
	return s.reasons
}
