package controllers

import (
	"context"
	"fmt"
	"time"

	"k8s.io/utils/ptr"

	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	"github.com/kyma-project/istio/operator/internal/status"

	"github.com/go-logr/logr"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
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
		It("Should fail to reconcile Istio CR in different than kyma-system namespace and set error state", func() {
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
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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
			result, err := istioController.Reconcile(context.TODO(), req)

			//then
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			updatedIstioCR := operatorv1alpha1.Istio{}
			err = client.Get(context.TODO(), types.NamespacedName{Name: "default"}, &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).To(Equal(operatorv1alpha1.Error))
			Expect(updatedIstioCR.Status.Description).To(Equal("Stopped Istio CR reconciliation: Istio CR is not in kyma-system namespace"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect((*updatedIstioCR.Status.Conditions)).To(HaveLen(1))
			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha1.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonReconcileFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		})

		It("Should not return an error when CR was not found", func() {
			// given
			apiClient := createFakeClient()

			sut := &IstioReconciler{
				Client:                 apiClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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

		It("Should call update status to processing when CR is not deleted", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
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
				proxySidecars:          &proxySidecarsReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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
					Finalizers: []string{"istios.operator.kyma-project.io/test-mock"},
				},
			}
			statusMock := NewStatusMock()
			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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

		It("Should return an error when update status to deleting failed", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
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
				proxySidecars:          &proxySidecarsReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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

		It("Should not requeue a CR without finalizers, because it's considered to be in deletion", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
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
				proxySidecars:          &proxySidecarsReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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

		It("Should set Ready status, update lastAppliedConfiguration annotation and requeue when successfully reconciled", func() {
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
						NumTrustedProxies: ptr.To(int(2)),
					},
				},
			}

			fakeClient := createFakeClient(istioCR)

			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result).Should(Equal(reconcile.Result{RequeueAfter: testReconciliationInterval}))

			updatedIstioCR := operatorv1alpha1.Istio{}
			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha1.Ready))
			Expect(updatedIstioCR.Annotations["operator.kyma-project.io/lastAppliedConfiguration"]).To(ContainSubstring("{\"config\":{\"numTrustedProxies\":2},"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect((*updatedIstioCR.Status.Conditions)).To(HaveLen(2))

			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha1.ConditionTypeProxySidecarRestartSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonProxySidecarRestartSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionTrue))

			Expect((*updatedIstioCR.Status.Conditions)[1].Type).To(Equal(string(operatorv1alpha1.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[1].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonReconcileSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[1].Status).To(Equal(metav1.ConditionTrue))
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
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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
			Expect(statusMock.updateConditionsCalled).Should(BeTrue())
			Expect(statusMock.GetConditions()).Should(Equal([]operatorv1alpha1.ConditionReasonWithMessage{
				operatorv1alpha1.NewConditionReasonWithMessage(operatorv1alpha1.ConditionReasonProxySidecarRestartSucceeded),
				operatorv1alpha1.NewConditionReasonWithMessage(operatorv1alpha1.ConditionReasonIngressGatewayReconcileSucceeded),
			}))
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
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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

			updatedIstioCR := operatorv1alpha1.Istio{}
			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha1.Error))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("test error description"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect((*updatedIstioCR.Status.Conditions)).To(HaveLen(1))

			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha1.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonIstioInstallUninstallFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
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
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
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

			updatedIstioCR := operatorv1alpha1.Istio{}
			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha1.Error))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Error occurred during reconciliation of Istio Sidecars"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect((*updatedIstioCR.Status.Conditions)).To(HaveLen(2))

			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha1.ConditionTypeProxySidecarRestartSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonProxySidecarRestartFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))

			Expect((*updatedIstioCR.Status.Conditions)[1].Type).To(Equal(string(operatorv1alpha1.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[1].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonReconcileFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[1].Status).To(Equal(metav1.ConditionFalse))
		})

		It("Should set ready status when successfully reconciled oldest Istio CR", func() {
			// given
			oldestIstioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:              istioCrName,
					Namespace:         testNamespace,
					UID:               "1",
					CreationTimestamp: metav1.Unix(1494505756, 0),
					Finalizers: []string{
						"istios.operator.kyma-project.io/istio-installation",
					},
				},
				Spec: operatorv1alpha1.IstioSpec{
					Config: operatorv1alpha1.Config{
						NumTrustedProxies: ptr.To(int(2)),
					},
				},
			}
			newerIstioCR := &operatorv1alpha1.Istio{
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
				ingressGateway:         &ingressGatewayReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).ShouldNot(HaveOccurred())
			Expect(result.RequeueAfter).Should(Equal(testReconciliationInterval))

			updatedIstioCR := operatorv1alpha1.Istio{}
			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(oldestIstioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha1.Ready))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect((*updatedIstioCR.Status.Conditions)).To(HaveLen(2))

			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha1.ConditionTypeProxySidecarRestartSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonProxySidecarRestartSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionTrue))

			Expect((*updatedIstioCR.Status.Conditions)[1].Type).To(Equal(string(operatorv1alpha1.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[1].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonReconcileSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[1].Status).To(Equal(metav1.ConditionTrue))
		})

		It("Should set an error status and do not requeue an Istio CR when an older Istio CR is present", func() {
			// given
			oldestIstioCR := &operatorv1alpha1.Istio{
				ObjectMeta: metav1.ObjectMeta{
					Name:              istioCrName,
					Namespace:         testNamespace,
					UID:               "1",
					CreationTimestamp: metav1.Unix(1494505756, 0),
				},
			}
			newerIstioCRName := fmt.Sprintf("%s-2", istioCrName)
			newerIstioCR := &operatorv1alpha1.Istio{
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
				proxySidecars:          &proxySidecarsReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: newerIstioCRName}})

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			updatedIstioCR := operatorv1alpha1.Istio{}
			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(newerIstioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha1.Error))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring(fmt.Sprintf("only Istio CR %s in %s reconciles the module", istioCrName, testNamespace)))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect((*updatedIstioCR.Status.Conditions)).To(HaveLen(1))

			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha1.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonOlderCRExists)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		})

		It("Should set an error status and requeue an Istio CR when is unable to list Istio CRs", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
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
				proxySidecars:          &proxySidecarsReconciliationMock{},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			updatedIstioCR := operatorv1alpha1.Istio{}
			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha1.Error))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Unable to list Istio CRs"))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect((*updatedIstioCR.Status.Conditions)).To(HaveLen(1))

			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha1.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonReconcileFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		})

		It("Should set a warning if warning happened during sidecars reconciliation", func() {
			// given
			istioCR := &operatorv1alpha1.Istio{
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
			sut := &IstioReconciler{
				Client:                 fakeClient,
				Scheme:                 getTestScheme(),
				istioInstallation:      &istioInstallationReconciliationMock{},
				proxySidecars:          &proxySidecarsReconciliationMock{warningMessage: "blah"},
				istioResources:         &istioResourcesReconciliationMock{},
				ingressGateway:         &ingressGatewayReconciliationMock{},
				log:                    logr.Discard(),
				statusHandler:          status.NewStatusHandler(fakeClient),
				reconciliationInterval: testReconciliationInterval,
			}

			// when
			result, err := sut.Reconcile(context.Background(), reconcile.Request{NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: istioCrName}})

			// then
			Expect(err).To(HaveOccurred())
			Expect(result.Requeue).To(BeFalse())

			updatedIstioCR := operatorv1alpha1.Istio{}
			err = fakeClient.Get(context.TODO(), client.ObjectKeyFromObject(istioCR), &updatedIstioCR)
			Expect(err).To(Not(HaveOccurred()))

			Expect(updatedIstioCR.Status.State).Should(Equal(operatorv1alpha1.Warning))
			Expect(updatedIstioCR.Status.Description).To(ContainSubstring("Not all pods with Istio injection could be restarted. Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning: Istio controller could not restart one or more istio-injected pods."))

			Expect(updatedIstioCR.Status.Conditions).ToNot(BeNil())
			Expect((*updatedIstioCR.Status.Conditions)).To(HaveLen(2))

			Expect((*updatedIstioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha1.ConditionTypeProxySidecarRestartSucceeded)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonProxySidecarManualRestartRequired)))
			Expect((*updatedIstioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))

			Expect((*updatedIstioCR.Status.Conditions)[1].Type).To(Equal(string(operatorv1alpha1.ConditionTypeReady)))
			Expect((*updatedIstioCR.Status.Conditions)[1].Reason).To(Equal(string(operatorv1alpha1.ConditionReasonReconcileFailed)))
			Expect((*updatedIstioCR.Status.Conditions)[1].Status).To(Equal(metav1.ConditionFalse))
		})
	})
})

type ingressGatewayReconciliationMock struct {
}

func (i *ingressGatewayReconciliationMock) AddReconcilePredicate(_ filter.IngressGatewayPredicate) Reconciliation {
	return i
}

func (i *ingressGatewayReconciliationMock) Reconcile(_ context.Context) described_errors.DescribedError {
	return nil
}

type istioResourcesReconciliationMock struct {
}

func (i *istioResourcesReconciliationMock) AddReconcileResource(_ istio_resources.Resource) istio_resources.ResourcesReconciliation {
	return i
}

func (i *istioResourcesReconciliationMock) Reconcile(_ context.Context, _ operatorv1alpha1.Istio) described_errors.DescribedError {
	return nil
}

type shouldFailClient struct {
	client.Client
	FailOnList bool
}

func (p *shouldFailClient) List(ctx context.Context, list client.ObjectList, _ ...client.ListOption) error {
	if p.FailOnList {
		return errors.New("intentionally failing client on list")
	}
	return p.Client.List(ctx, list)
}

type istioInstallationReconciliationMock struct {
	err described_errors.DescribedError
}

func (i *istioInstallationReconciliationMock) Reconcile(_ context.Context, istioCR *operatorv1alpha1.Istio, statusHandler status.Status, _ string) described_errors.DescribedError {
	return i.err
}

type proxySidecarsReconciliationMock struct {
	warningMessage string
	err            error
}

func (p *proxySidecarsReconciliationMock) AddReconcilePredicate(_ filter.SidecarProxyPredicate) {
}

func (p *proxySidecarsReconciliationMock) Reconcile(_ context.Context, _ operatorv1alpha1.Istio) (string, error) {
	return p.warningMessage, p.err
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
	conditionsError           error
	updateConditionsCalled    bool
	conditionReasons          []operatorv1alpha1.ConditionReasonWithMessage
}

func NewStatusMock() *StatusMock {
	return &StatusMock{
		conditionReasons: []operatorv1alpha1.ConditionReasonWithMessage{},
	}
}

func (s *StatusMock) UpdateToProcessing(_ context.Context, _ *operatorv1alpha1.Istio) error {
	s.updatedToProcessingCalled = true
	return s.processingError
}

func (s *StatusMock) UpdateToDeleting(_ context.Context, _ *operatorv1alpha1.Istio) error {
	s.updatedToDeletingCalled = true
	return s.deletingError
}

func (s *StatusMock) UpdateToReady(_ context.Context, _ *operatorv1alpha1.Istio) error {
	s.updatedToReadyCalled = true
	return s.readyError
}

func (s *StatusMock) UpdateToError(_ context.Context, _ *operatorv1alpha1.Istio, _ described_errors.DescribedError, conditionReasons ...operatorv1alpha1.ConditionReasonWithMessage) error {
	s.updatedToErrorCalled = true
	s.conditionReasons = append(s.conditionReasons, conditionReasons...)
	return s.errorError
}

func (s *StatusMock) UpdateConditions(_ context.Context, _ *operatorv1alpha1.Istio, conditionReasons ...operatorv1alpha1.ConditionReasonWithMessage) error {
	s.updateConditionsCalled = true
	s.conditionReasons = append(s.conditionReasons, conditionReasons...)
	return s.conditionsError
}

func (s *StatusMock) GetConditions() []operatorv1alpha1.ConditionReasonWithMessage {
	return s.conditionReasons
}
