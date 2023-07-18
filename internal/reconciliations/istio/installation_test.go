package istio_test

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/kyma-project/istio/operator/pkg/lib/annotations"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

const (
	istioVersion         string = "1.16.1"
	istioImageBase       string = "distroless"
	resourceListPath     string = "test/test_controlled_resource_list.yaml"
	testKey              string = "key"
	testValue            string = "value"
	istioDisclaimerKey   string = "istios.operator.kyma-project.io/managed-by-disclaimer"
	istioDisclaimerValue string = "DO NOT EDIT - This resource is managed by Kyma.\nAny modifications are discarded and the resource is reverted to the original state."
)

var istioTag = fmt.Sprintf("%s-%s", istioVersion, istioImageBase)

var _ = Describe("Installation reconciliation", func() {

	It("should reconcile when Istio CR and Istio version didn't change", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should install and update Istio CR status when Istio is not installed", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  status.NewDefaultStatusHandler(),
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))
	})

	It("should label and annotate istio-system namespace after Istio installation without overriding existing labels and annotations", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())

		ns := corev1.Namespace{}
		_ = c.Get(context.TODO(), types.NamespacedName{Name: "istio-system"}, &ns)
		Expect(ns.Labels).To(HaveKeyWithValue(testKey, testValue))
		Expect(ns.Annotations).To(HaveKeyWithValue(testKey, testValue))
		Expect(ns.Labels).To(HaveKeyWithValue("namespaces.warden.kyma-project.io/validate", "enabled"))
		Expect(ns.Annotations).To(HaveKeyWithValue(istioDisclaimerKey, istioDisclaimerValue))
	})

	It("should fail if after install and update Istio pods do not match target version", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.0")
		istioNamespace := createNamespace("istio-system")
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr, istiod, istioNamespace),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  status.NewDefaultStatusHandler(),
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("istio-system pods version: 1.16.0 do not match target version: 1.16.1"))
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))
	})

	It("should add installation finalizer when Istio is installed", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations:     map[string]string{},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(returnedIstioCr.Finalizers).To(ContainElement("istios.operator.kyma-project.io/istio-installation"))
	})

	It("should execute install to upgrade istio and update Istio CR status when NumTrustedProxies has changed and restart Istio GW", func() {
		// given

		newNumTrustedProxies := 3
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"NumTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &newNumTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  status.NewDefaultStatusHandler(),
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))

		igwDeployment = &appsv1.Deployment{}
		error := c.Get(context.TODO(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, igwDeployment)
		Expect(error).To(Not(HaveOccurred()))

		hasRestartAnnotation := annotations.HasRestartAnnotation(igwDeployment.Spec.Template.Annotations)
		Expect(hasRestartAnnotation).To(BeTrue())
	})

	It("should execute install to upgrade istio and update Istio CR status when NumTrustedProxies has not changed and do not restart Istio GW", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  status.NewDefaultStatusHandler(),
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))

		currentIGWDeployment := appsv1.Deployment{}
		error := c.Get(context.TODO(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, &currentIGWDeployment)

		Expect(error).To(Not(HaveOccurred()))
		Expect(currentIGWDeployment.Spec.Template.Annotations["reconciler.kyma-project.io/lastRestartDate"]).To(BeEmpty())
	})

	It("should update Istio CR status to error when NumTrustedProxies has changed and fails to restart Istio GW", func() {
		// given

		newNumTrustedProxies := 3
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"NumTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &newNumTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "istio-system",
				Name:      "istio-ingressgateway",
				Annotations: map[string]string{
					"failAnnotation": "intentional",
				},
			},
		}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)
		shouldFailClient := &shouldFailFakeClientOnAnnotation{c, "failAnnotation"}
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         shouldFailClient,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  status.NewDefaultStatusHandler(),
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))

		igwDeployment = &appsv1.Deployment{}
		error := c.Get(context.TODO(), types.NamespacedName{Namespace: "istio-system", Name: "istio-ingressgateway"}, igwDeployment)
		Expect(error).To(Not(HaveOccurred()))

		hasRestartAnnotation := annotations.HasRestartAnnotation(igwDeployment.Spec.Template.Annotations)
		Expect(hasRestartAnnotation).To(BeFalse())
	})

	It("should execute install to upgrade istio and update Istio CR status when Istio version has changed", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.17.0")
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   "1.17.0",
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  status.NewDefaultStatusHandler(),
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))
	})

	It("should not execute install to downgrade istio", func() {
		// given

		istioVersionDowngrade := "1.16.0"
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr, istiod),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersionDowngrade,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("target Istio version (1.16.0-distroless) is lower than current version (1.16.1-distroless) - downgrade not supported"))
		Expect(err.Description()).To(Equal("Istio version check failed: target Istio version (1.16.0-distroless) is lower than current version (1.16.1-distroless) - downgrade not supported"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should not execute install to upgrade istio from 1.16.1 to 1.18.0", func() {
		// given

		istioVersionTwoMinor := "1.18.0"
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr, istiod),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersionTwoMinor,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("target Istio version (1.18.0-distroless) is higher than current Istio version (1.16.1-distroless) - the difference between versions exceed one minor version"))
		Expect(err.Description()).To(Equal("Istio version check failed: target Istio version (1.18.0-distroless) is higher than current Istio version (1.16.1-distroless) - the difference between versions exceed one minor version"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should not execute install to upgrade istio from 1.16.1 to 2.0.0", func() {
		// given

		istioVersionOneMajor := "2.0.0"
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr, istiod),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersionOneMajor,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("target Istio version (2.0.0-distroless) is different than current Istio version (1.16.1-distroless) - major version upgrade is not supported"))
		Expect(err.Description()).To(Equal("Istio version check failed: target Istio version (2.0.0-distroless) is different than current Istio version (1.16.1-distroless) - major version upgrade is not supported"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should fail when istio version is invalid", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.17.0")
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   "fake",
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("fake is not in dotted-tri format"))
		Expect(err.Description()).To(Equal("Istio version check failed: fake is not in dotted-tri format"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should fail when istio sidecar version on cluster is invalid", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "fake")
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   "1.17.0",
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("Invalid Semantic Version"))
		Expect(err.Description()).To(Equal("Could not get Istio sidecar version on cluster: Invalid Semantic Version"))
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should fail when custom resource list merging fails", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "fake")
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   "1.17.0",
			IstioImageBase: istioImageBase,
			Merger: MergerMock{
				mergeError: errors.New("merging failed"),
			},
			StatusHandler: StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("merging failed"))
		Expect(err.Description()).To(Equal("Could not get configuration from Istio Operator file: merging failed"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should fail if status update to processing fails", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.17.0")
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   "1.17.0",
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler: StatusMock{
				processingError: errors.New("update error"),
			},
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("update error"))
		Expect(err.Description()).To(Equal("Could not set status to processing: update error"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should fail when istio installation fails", func() {
		// given

		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.17.0")
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{
			installError: errors.New("installation failed"),
		}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   "1.17.0",
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("installation failed"))
		Expect(err.Description()).To(Equal("Could not install Istio: installation failed"))
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should not install or uninstall when Istio CR has changed, but has deletion timestamp", func() {
		// given
		now := metav1.NewTime(time.Now())
		newNumTrustedProxies := 3
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &newNumTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
	})

	It("should uninstall when Istio CR has deletion timestamp and installation finalizer", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeTrue())
	})

	It("should fail when uninstall fails", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		mockClient := mockLibraryClient{
			uninstallError: errors.New("uninstall failed"),
		}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("uninstall failed"))
		Expect(err.Description()).To(Equal("Could not uninstall istio: uninstall failed"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeTrue())
	})

	It("should install but not uninstall when Istio CR has no deletion timestamp", func() {
		// given
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		igwDeployment := &appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: "istio-system", Name: "istio-ingressgateway"}}
		c := createFakeClient(&istioCr, istiod, istioNamespace, igwDeployment)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should not uninstall when Istio CR has deletion timestamp but no installation finalizer", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should uninstall if there are only default Istio resources present", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", istioVersion)
		istioNamespace := createNamespace("istio-system")
		c := createFakeClient(&istioCr, istiod, istioNamespace, &networkingv1alpha3.EnvoyFilter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-default-resource",
				Namespace: "istio-system",
			},
		})

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeTrue())
	})

	It("should return an error when update to status deleting fails", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client: createFakeClient(&istioCr, &networkingv1alpha3.EnvoyFilter{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-default-resource",
					Namespace: "istio-system",
				},
			}),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler: StatusMock{
				deletingError: errors.New("could not update status to deleting"),
			},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("could not update status to deleting"))
		Expect(err.Description()).To(Equal("Could not set status to deleting: could not update status to deleting"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should not uninstall if there are Istio resources present", func() {
		// given
		now := metav1.NewTime(time.Now())
		numTrustedProxies := 1
		istioCr := operatorv1alpha1.Istio{ObjectMeta: metav1.ObjectMeta{
			Name:            "default",
			ResourceVersion: "1",
			Annotations: map[string]string{
				istio.LastAppliedConfiguration: fmt.Sprintf(`{"config":{"numTrustedProxies":%d},"IstioTag":"%s"}`, numTrustedProxies, istioTag),
			},
			DeletionTimestamp: &now,
			Finalizers:        []string{"istios.operator.kyma-project.io/istio-installation"},
		},
			Spec: operatorv1alpha1.IstioSpec{
				Config: operatorv1alpha1.Config{
					NumTrustedProxies: &numTrustedProxies,
				},
			},
		}

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client: createFakeClient(&istioCr, &networkingv1alpha3.VirtualService{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mock-vs",
					Namespace: "mock-ns",
				},
			}),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
			StatusHandler:  StatusMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("could not delete Istio module instance since there are 1 customer resources present"))
		Expect(err.Description()).To(Equal("Resources blocking deletion: VirtualService:mock-ns/mock-vs"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
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
	err := operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = networkingv1alpha3.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(objects...).Build()
}

type shouldFailFakeClientOnAnnotation struct {
	client.Client
	failAnnotation string
}

func (p *shouldFailFakeClientOnAnnotation) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	_, found := obj.GetAnnotations()[p.failAnnotation]
	if found {
		return fmt.Errorf("Intentionally failing client update call on annotation: %s", p.failAnnotation)
	}
	return p.Client.Update(ctx, obj)
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
}

func (m MergerMock) Merge(_ string, _ *operatorv1alpha1.Istio, _ manifest.TemplateData, _ clusterconfig.ClusterConfiguration) (string, error) {
	return "mocked istio operator merge result", m.mergeError
}

func (m MergerMock) GetIstioOperator(_ string) (istioOperator.IstioOperator, error) {
	return istioOperator.IstioOperator{}, m.getIstioOperatorError
}

func (m MergerMock) SetIstioInstallFlavor(_ clusterconfig.ClusterSize) {}

type StatusMock struct {
	result          ctrl.Result
	processingError error
	readyError      error
	deletingError   error
	errorError      error
}

func (s StatusMock) SetProcessing(_ context.Context, _ string, _ client.Client, _ *operatorv1alpha1.Istio, _ metav1.Condition, _ ...time.Duration) (ctrl.Result, error) {
	return s.result, s.processingError
}

func (s StatusMock) SetReady(_ context.Context, _ client.Client, _ *operatorv1alpha1.Istio, _ metav1.Condition, _ ...time.Duration) (ctrl.Result, error) {
	return s.result, s.readyError
}

func (s StatusMock) SetError(_ context.Context, _ described_errors.DescribedError, _ client.Client, _ *operatorv1alpha1.Istio, _ metav1.Condition, _ ...time.Duration) (ctrl.Result, error) {
	return s.result, s.errorError
}

func (s StatusMock) SetDeleting(_ context.Context, _ client.Client, _ *operatorv1alpha1.Istio, _ metav1.Condition, _ ...time.Duration) (ctrl.Result, error) {
	return s.result, s.deletingError
}
