package proxy_test

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/manifest"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"testing"

	"github.com/go-logr/logr"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/proxy"
	"github.com/kyma-project/istio/operator/pkg/lib/gatherer"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

	It("should fail proxy reset if Istio pods do not match target version", func() {
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
		sidecars := proxy.Sidecars{
			Log:            logr.Discard(),
			Client:         createFakeClient(&istioCr, istiod),
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			Merger:         MergerMock{},
		}
		// when
		err := sidecars.Reconcile(context.TODO(), istioCr)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("istio-system pods version: 1.16.0 do not match target version: 1.16.1"))
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
	return istioOperator.IstioOperator{}, nil
}

func (m MergerMock) SetIstioInstallFlavor(_ clusterconfig.ClusterSize) {}
