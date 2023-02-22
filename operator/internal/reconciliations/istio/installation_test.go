package istio_test

import (
	"context"
	"fmt"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"time"
)

const (
	istioVersion             string = "1.16.1"
	istioImageBase           string = "distroless"
	defaultIstioOperatorPath        = "test/test-operator.yaml"
	workingDir                      = "/tmp"
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
			Client:         &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
		}

		// when
		_, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

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
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

		// then
		Expect(err).ShouldNot(HaveOccurred())
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
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

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
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         &mockClient,
			IstioVersion:   "1.17.0",
			IstioImageBase: istioImageBase,
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))
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
		mockClient := mockLibraryClient{}
		installation := istio.Installation{
			Client:         &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeTrue())
		Expect(mockClient.uninstallCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).To(Equal(operatorv1alpha1.Processing))
	})

	It("should not install when Istio CR has changed, but has deletion timestamp", func() {
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
			Client:         &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
		}
		// when
		returnedIstioCr, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

		// then
		Expect(err).ShouldNot(HaveOccurred())
		Expect(mockClient.installCalled).To(BeFalse())
		Expect(returnedIstioCr.Status.State).NotTo(Equal(operatorv1alpha1.Processing))
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
			Client:         &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
		}

		// when
		_, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

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
			Client:         &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
		}

		// when
		_, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

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
			Client:         &mockClient,
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
		}

		// when
		_, err := installation.Reconcile(context.TODO(), createFakeClient(istioCr), istioCr, defaultIstioOperatorPath, workingDir)

		// then
		Expect(err).ShouldNot(HaveOccurred())
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

func createFakeClient(istioCr operatorv1alpha1.Istio) client.Client {
	err := operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(&istioCr).Build()
}
