package proxy_test

import (
	"context"
	"os"
	"testing"

	"github.com/go-logr/logr"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/internal/reconciliations/proxy"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

const (
	istioVersion   = "1.16.1"
	istioImageBase = "distroless"
)

func TestProxies(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Merge Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("merge-api-suite", report)
})

var _ = Describe("Sidecars reconciliation", func() {
	It("Should fail proxy reset if Istio pods do not match target version", func() {
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
		sidecars := proxy.NewReconciler(istioVersion, istioImageBase, logr.Discard(), createFakeClient(&istioCr, istiod),
			&MergerMock{}, sidecars.NewProxyResetter(), []filter.SidecarProxyPredicate{})
		// when
		warningMessage, err := sidecars.Reconcile(context.TODO(), istioCr)

		// then
		Expect(warningMessage).To(Equal(""))
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("istio-system pods version: 1.16.0 do not match target version: 1.16.1"))
	})

	It("Should succeed proxy reset even if more than 5 proxies could not be reset and will return a warning", func() {
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
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.1")
		proxyResetter := &proxyResetterMock{
			restartWarnings: []restart.RestartWarning{
				{
					Name:      "name1",
					Namespace: "ns1",
				},
				{
					Name:      "name2",
					Namespace: "ns2",
				},
				{
					Name:      "name3",
					Namespace: "ns3",
				},
				{
					Name:      "name4",
					Namespace: "ns4",
				},
				{
					Name:      "name5",
					Namespace: "ns5",
				},
				{
					Name:      "name6",
					Namespace: "ns6",
				},
			},
		}
		sidecars := proxy.NewReconciler(istioVersion, istioImageBase, logr.Discard(), createFakeClient(&istioCr, istiod),
			&MergerMock{}, proxyResetter, []filter.SidecarProxyPredicate{})

		// when
		warningMessage, err := sidecars.Reconcile(context.TODO(), istioCr)

		// then
		Expect(warningMessage).To(Equal("The sidecars of the following workloads could not be restarted: ns1/name1, ns2/name2, ns3/name3, ns4/name4, ns5/name5 and 1 additional workload(s)"))
		Expect(err).ShouldNot(HaveOccurred())
	})

	It("Should succeed proxy reset even if less than 5 proxies could not be reset and will return a warning", func() {
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
		istiod := createPod("istiod", gatherer.IstioNamespace, "discovery", "1.16.1")
		proxyResetter := &proxyResetterMock{
			restartWarnings: []restart.RestartWarning{
				{
					Name:      "name1",
					Namespace: "ns1",
				},
				{
					Name:      "name2",
					Namespace: "ns2",
				},
			},
		}
		sidecars := proxy.NewReconciler(istioVersion, istioImageBase, logr.Discard(), createFakeClient(&istioCr, istiod),
			&MergerMock{}, proxyResetter, []filter.SidecarProxyPredicate{})

		// when
		warningMessage, err := sidecars.Reconcile(context.TODO(), istioCr)

		// then
		Expect(warningMessage).To(Equal("The sidecars of the following workloads could not be restarted: ns1/name1, ns2/name2"))
		Expect(err).ShouldNot(HaveOccurred())
	})
})

func createFakeClient(objects ...client.Object) client.Client {
	err := operatorv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
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

type MergerMock struct {
}

func (m MergerMock) Merge(_ string, _ *operatorv1alpha1.Istio, _ manifest.TemplateData, _ clusterconfig.ClusterConfiguration) (string, error) {
	return "mocked istio operator merge result", nil
}

func (m MergerMock) GetIstioOperator(_ string) (istioOperator.IstioOperator, error) {
	iop := istioOperator.IstioOperator{}
	manifest, err := os.ReadFile("../../../manifests/istio-operator-template.yaml")
	if err == nil {
		err = yaml.Unmarshal(manifest, &iop)
	}
	return iop, err
}

func (m MergerMock) SetIstioInstallFlavor(_ clusterconfig.ClusterSize) {}

type proxyResetterMock struct {
	err             error
	restartWarnings []restart.RestartWarning
}

func (p *proxyResetterMock) ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, logger *logr.Logger) ([]restart.RestartWarning, error) {
	return p.restartWarnings, p.err
}
