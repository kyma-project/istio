package istio_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/pkg/labels"
	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	"time"

	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/resources"
	"github.com/kyma-project/istio/operator/internal/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/pkg/errors"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

const (
	istioVersion string = "1.16.1"
	istioTag            = istioVersion + "-distroless"
	testKey      string = "key"
	testValue    string = "value"
)

var _ = Describe("Installation reconciliation", func() {
	It("should reconcile when Istio CR and Istio version didn't change", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
		Expect(*istioCR.Status.Conditions).To(HaveLen(1))
		Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
		Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioInstallSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
	})

	It("should install and update Istio CR status when Istio is not installed", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
			Status: operatorv1alpha2.IstioStatus{
				State: operatorv1alpha2.Processing,
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(istioCR.Status.State).To(Equal(operatorv1alpha2.Processing))

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
		Expect(*istioCR.Status.Conditions).To(HaveLen(1))
		Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
		Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioInstallSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
	})

	It("should label and annotate istio-system namespace after Istio installation without overriding existing labels and annotations", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())

		ns := corev1.Namespace{}
		_ = c.Get(context.Background(), types.NamespacedName{Name: "istio-system"}, &ns)
		Expect(ns.Labels).To(HaveKeyWithValue(testKey, testValue))
		Expect(ns.Annotations).To(HaveKeyWithValue(testKey, testValue))
		Expect(ns.Labels).To(HaveKeyWithValue("namespaces.warden.kyma-project.io/validate", "enabled"))
		Expect(ns.Annotations).To(HaveKeyWithValue(resources.DisclaimerKey, resources.DisclaimerValue))

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
	})

	It("should fail if after install and update Istio pods do not match target version", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
			Status: operatorv1alpha2.IstioStatus{
				State: operatorv1alpha2.Processing,
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.0")
		istioNamespace := createNamespace("istio-system")
		mockClient := mockLibraryClient{}
		c := createFakeClient(&istioCR, istiod, istioNamespace)
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("istio-system Pods version 1.16.0 do not match istio operator version 1.16.1"))
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(istioCR.Status.State).To(Equal(operatorv1alpha2.Processing))

		Expect(istioCR.Status.Conditions).To(BeNil())
	})

	It("should add installation finalizer when Istio is installed", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		mockClient := mockLibraryClient{}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment)
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(istioCR.Finalizers).To(ContainElement("istios.operator.kyma-project.io/istio-installation"))

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
	})

	It("should execute install to upgrade istio and update Istio CR status when Istio version has changed", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
			Status: operatorv1alpha2.IstioStatus{
				State: operatorv1alpha2.Processing,
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.17.0")
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: "1.17.0-distroless"},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(istioCR.Status.State).To(Equal(operatorv1alpha2.Processing))
		Expect(istioCR.Status.Conditions).ToNot(BeNil())
	})

	It("should execute install when only Istio image type has changed to debug", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
			Status: operatorv1alpha2.IstioStatus{
				State: operatorv1alpha2.Processing,
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion+"-debug")
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioVersion + "-debug"},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(istioCR.Status.State).To(Equal(operatorv1alpha2.Processing))
	})

	DescribeTable("updating Istio version",
		func(mergerIstioVersionTag string, expectedErrorMessage string) {
			// given
			numTrustedProxies := 1
			istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
				Name:            "default",
				ResourceVersion: "1",
				Annotations: map[string]string{
					labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
				},
			},
				Spec: operatorv1alpha2.IstioSpec{
					Config: operatorv1alpha2.Config{
						NumTrustedProxies: &numTrustedProxies,
					},
				},
			}

			istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
			mockClient := mockLibraryClient{}
			c := createFakeClient(&istioCR, istiod)
			installation := istio.Installation{
				Client:      c,
				IstioClient: &mockClient,
				Merger:      MergerMock{tag: mergerIstioVersionTag},
			}
			statusHandler := status.NewStatusHandler(c)

			// when
			_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

			// then
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).To(Equal(expectedErrorMessage))
			Expect(err.Description()).To(ContainSubstring("Istio version update is not allowed"))
			Expect(err.Description()).To(ContainSubstring(expectedErrorMessage))
			Expect(err.ShouldSetCondition()).To(BeFalse())
			Expect(err.Level()).To(Equal(described_errors.Warning))
			Expect(mockClient.installCalled).To(BeFalse())
			Expect(mockClient.uninstallCalled).To(BeFalse())

			Expect(istioCR.Status.Conditions).ToNot(BeNil())
			Expect(*istioCR.Status.Conditions).To(HaveLen(1))
			Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
			Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioVersionUpdateNotAllowed)))
			Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
		},
		Entry("should return warning and not execute install when new Istio version is lower", "1.16.0-distroless", "target Istio version (1.16.0-distroless) is lower than current version (1.16.1-distroless) - downgrade not supported"),
		Entry("should return warning and not execute install when new Istio version is more than one minor version higher", "1.18.0-distroless", "target Istio version (1.18.0-distroless) is higher than current Istio version (1.16.1-distroless) - the difference between versions exceed one minor version"),
		Entry("should return warning and not execute install when new Istio version has a higher major version", "2.0.0-distroless", "target Istio version (2.0.0-distroless) is different than current Istio version (1.16.1-distroless) - major version upgrade is not supported"),
	)

	It("should fail when istio version is invalid", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.17.0")
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCR, istiod, istioNamespace)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: "fake-distroless"},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("fake is not in dotted-tri format"))
		Expect(err.Description()).To(Equal("Could not get Istio version from istio operator file: fake is not in dotted-tri format"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(istioCR.Status.Conditions).To(BeNil())
	})

	It("should fail when custom resource list merging fails", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "fake")
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCR, istiod, istioNamespace)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger: MergerMock{
				mergeError: errors.New("merging failed"),
				tag:        "1.17.0-distroless",
			},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("merging failed"))
		Expect(err.Description()).To(Equal("Could not merge Istio operator configuration: merging failed"))
		Expect(err.ShouldSetCondition()).To(BeFalse())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
		Expect(*istioCR.Status.Conditions).To(HaveLen(1))
		Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
		Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonCustomResourceMisconfigured)))
		Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
	})

	It("should fail when istio installation fails", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.17.0")
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCR, istiod, istioNamespace)
		mockClient := mockLibraryClient{
			installError: errors.New("installation failed"),
		}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: "1.17.0-distroless"},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("installation failed"))
		Expect(err.Description()).To(Equal("Could not install Istio: installation failed"))
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(istioCR.Status.Conditions).To(BeNil())
	})

	It("should not install or uninstall when Istio CR has changed, but has deletion timestamp", func() {
		// given
		now := metav1.NewTime(time.Now())
		newNumTrustedProxies := 3
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":1},"IstioTag":"%s"}`, istioTag),
			},
			DeletionTimestamp: &now,
			// We need to add a dummy finalizer to be able to set the deletion timestamp.
			Finalizers: []string{"istios.operator.kyma-project.io/test-mock"},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &newNumTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
		Expect(*istioCR.Status.Conditions).To(HaveLen(1))
		Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
		Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioInstallNotNeeded)))
		Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
	})

	It("should uninstall when Istio CR has deletion timestamp and installation finalizer", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCR, istiod, istioNamespace)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeTrue())

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
		Expect(*istioCR.Status.Conditions).To(HaveLen(1))
		Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
		Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioUninstallSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
	})

	It("should fail when uninstall fails", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		mockClient := mockLibraryClient{
			uninstallError: errors.New("uninstall failed"),
		}
		c := createFakeClient(&istioCR)
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("uninstall failed"))
		Expect(err.Description()).To(Equal("Could not uninstall istio: uninstall failed"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeTrue())
		Expect(istioCR.Status.Conditions).To(BeNil())
	})

	It("should install but not uninstall when Istio CR has no deletion timestamp", func() {
		// given
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(istioCR.Status.Conditions).ToNot(BeNil())
	})

	It("should not uninstall when Istio CR has deletion timestamp but no installation finalizer", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(istiod, istioNamespace)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
		Expect(*istioCR.Status.Conditions).To(HaveLen(1))
		Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
		Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioInstallNotNeeded)))
		Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
	})

	It("should uninstall if there are only default Istio resources present", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCR, istiod, istioNamespace, &networkingv1alpha3.EnvoyFilter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "stats-filter-1.10",
				Namespace: "istio-system",
			},
		})
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeTrue())

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
		Expect(*istioCR.Status.Conditions).To(HaveLen(1))
		Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
		Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioUninstallSucceeded)))
		Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
	})

	It("should not uninstall if there are Istio resources present", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				labels.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		mockClient := mockLibraryClient{}
		c := createFakeClient(&istioCR, &networkingv1.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mock-vs",
				Namespace: "mock-ns",
			},
		})
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("could not delete Istio module instance since there are 1 customer resources present"))
		Expect(err.Description()).To(Equal("There are Istio resources that block deletion. Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning"))
		Expect(err.Level()).To(Equal(described_errors.Warning))
		Expect(err.ShouldSetCondition()).To(BeFalse())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())

		Expect(istioCR.Status.Conditions).ToNot(BeNil())
		Expect(*istioCR.Status.Conditions).To(HaveLen(1))
		Expect((*istioCR.Status.Conditions)[0].Type).To(Equal(string(operatorv1alpha2.ConditionTypeReady)))
		Expect((*istioCR.Status.Conditions)[0].Reason).To(Equal(string(operatorv1alpha2.ConditionReasonIstioCRsDangling)))
		Expect((*istioCR.Status.Conditions)[0].Status).To(Equal(metav1.ConditionFalse))
	})

	It("should have all istio components labeled with kyma-project.io/module=istio label", func() {
		numTrustedProxies := 1
		istioCR := operatorv1alpha2.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha2.IstioSpec{
				Config: operatorv1alpha2.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway", Labels: map[string]string{"operator.istio.io/component": "IngressGateways"}}}
		istioDaemonSet := &appsv1.DaemonSet{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-cni-node", Labels: map[string]string{"operator.istio.io/component": "Cni"}}}
		istioConfigMap := &corev1.ConfigMap{ObjectMeta: v1.ObjectMeta{Name: "istio", Namespace: "istio-system", Labels: map[string]string{"operator.istio.io/component": "Pilot"}}}
		c := createFakeClient(&istioCR, istiod, istioNamespace, igwDeployment, istioDaemonSet, istioConfigMap)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:      c,
			IstioClient: &mockClient,
			Merger:      MergerMock{tag: istioTag},
		}
		statusHandler := status.NewStatusHandler(c)

		// when
		_, err := installation.Reconcile(context.Background(), &istioCR, statusHandler)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())

		cm := corev1.ConfigMap{}
		cerr := c.Get(context.Background(), types.NamespacedName{Namespace: "istio-system", Name: "istio"}, &cm)
		Expect(cerr).ToNot(HaveOccurred())
		Expect(cm.Labels).To(HaveKeyWithValue("kyma-project.io/module", "istio"))

		d := appsv1.Deployment{}
		cerr = c.Get(context.Background(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, &d)
		Expect(cerr).ToNot(HaveOccurred())
		Expect(d.Labels).To(HaveKeyWithValue("kyma-project.io/module", "istio"))
		Expect(d.Spec.Template.Labels).To(HaveKeyWithValue("kyma-project.io/module", "istio"))

		ds := appsv1.DaemonSet{}
		cerr = c.Get(context.Background(), types.NamespacedName{Namespace: "istio-system", Name: "istio-cni-node"}, &ds)
		Expect(cerr).ToNot(HaveOccurred())
		Expect(ds.Labels).To(HaveKeyWithValue("kyma-project.io/module", "istio"))
		Expect(ds.Spec.Template.Labels).To(HaveKeyWithValue("kyma-project.io/module", "istio"))
	})
})

type mockLibraryClient struct {
	installCalled   bool
	uninstallCalled bool
	*istio.IstioClient
	installError   error
	uninstallError error
}

func (c *mockLibraryClient) Install(_ string) error {
	c.installCalled = true
	return c.installError
}

func (c *mockLibraryClient) Uninstall(_ context.Context) error {
	c.uninstallCalled = true
	return c.uninstallError
}

func createFakeClient(objects ...client.Object) client.Client {
	err := operatorv1alpha2.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1alpha3.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).WithStatusSubresource(objects...).Build()
}

func createPod(name, namespace, containerName, imageVersion string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  containerName,
					Image: "image:" + imageVersion,
				},
			},
		},
	}
}

func createNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      map[string]string{testKey: testValue},
			Annotations: map[string]string{testKey: testValue},
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Namespace",
			APIVersion: "v1",
		},
	}
}

type MergerMock struct {
	mergeError            error
	getIstioOperatorError error
	tag                   string
}

func (m MergerMock) Merge(_ clusterconfig.ClusterSize, _ *operatorv1alpha2.Istio, _ clusterconfig.ClusterConfiguration) (string, error) {
	return "mocked istio operator merge result", m.mergeError
}

func (m MergerMock) GetIstioOperator(_ clusterconfig.ClusterSize) (iopv1alpha1.IstioOperator, error) {
	iop := iopv1alpha1.IstioOperator{
		Spec: iopv1alpha1.IstioOperatorSpec{
			Tag: structpb.NewStringValue(m.tag),
		},
	}
	return iop, m.getIstioOperatorError
}

func (m MergerMock) GetIstioImageVersion() (istiooperator.IstioImageVersion, error) {
	return istiooperator.NewIstioImageVersionFromTag(m.tag)
}

func (m MergerMock) SetIstioInstallFlavor(_ clusterconfig.ClusterSize) {}
