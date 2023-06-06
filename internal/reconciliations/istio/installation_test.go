package istio_test

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/manifest"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"time"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	It("should not reconcile when Istio CR and Istio version didn't change", func() {
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

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
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
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))

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
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr, istiod, istioNamespace),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(returnedIstioCr.Finalizers).To(ContainElement("istios.operator.kyma-project.io/istio-installation"))
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
		c := createFakeClient(&istioCr, istiod, istioNamespace)

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         c,
			IstioClient:    &mockClient,
			IstioVersion:   "1.17.0",
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))

		ns := corev1.Namespace{}
		_ = c.Get(context.TODO(), types.NamespacedName{Name: "istio-system"}, &ns)
		Expect(ns.Labels).To(HaveKeyWithValue(testKey, testValue))
		Expect(ns.Annotations).To(HaveKeyWithValue(testKey, testValue))
		Expect(ns.Labels).To(HaveKeyWithValue("namespaces.warden.kyma-project.io/validate", "enabled"))
		Expect(ns.Annotations).To(HaveKeyWithValue(istioDisclaimerKey, istioDisclaimerValue))
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
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("target Istio version (1.16.0-distroless) is lower than current version (1.16.1-distroless) - downgrade not supported"))
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
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("target Istio version (1.18.0-distroless) is higher than current Istio version (1.16.1-distroless) - the difference between versions exceed one minor version"))
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
		}
		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("target Istio version (2.0.0-distroless) is different than current Istio version (1.16.1-distroless) - major version upgrade is not supported"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})

	It("should execute install to upgrade istio and update Istio CR status when Istio CR has changed", func() {
		// given

		newNumTrustedProxies := 3
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
					NumTrustedProxies: &newNumTrustedProxies,
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
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))

		ns := corev1.Namespace{}
		_ = c.Get(context.TODO(), types.NamespacedName{Name: "istio-system"}, &ns)
		Expect(ns.Labels).To(HaveKeyWithValue(testKey, testValue))
		Expect(ns.Annotations).To(HaveKeyWithValue(testKey, testValue))
		Expect(ns.Labels).To(HaveKeyWithValue("namespaces.warden.kyma-project.io/validate", "enabled"))
		Expect(ns.Annotations).To(HaveKeyWithValue(istioDisclaimerKey, istioDisclaimerValue))
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

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
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

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeTrue())
	})

	It("should not uninstall when Istio CR has no deletion timestamp", func() {
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

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
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

		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         createFakeClient(&istioCr),
			IstioClient:    &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
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
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeTrue())
	})

	It("should not uninstall if there are not default Istio resources present", func() {
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
		}

		// when
		_, err := installation.Reconcile(context.TODO(), istioCr, resourceListPath)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("could not delete Istio module instance since there are 1 customer created resources present"))
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(mockClient.uninstallCalled).To(BeFalse())
	})
})

type mockLibraryClient struct {
	installCalled   bool
	uninstallCalled bool
	*istio.IstioClient
}

func (c *mockLibraryClient) Install(_ string) error {
	c.installCalled = true
	return nil
}

func (c *mockLibraryClient) Uninstall(_ context.Context) error {
	c.uninstallCalled = true
	return nil
}

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
}

func (m MergerMock) Merge(_ *operatorv1alpha1.Istio, _ manifest.TemplateData, _ clusterconfig.ClusterConfiguration) (string, error) {
	return "mocked istio operator merge result", nil
}

func (m MergerMock) GetIstioOperator() (istioOperator.IstioOperator, error) {
	return istioOperator.IstioOperator{}, nil
}
