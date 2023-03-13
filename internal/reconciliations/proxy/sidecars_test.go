package proxy_test

import (
	"context"

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
	istioVersion             = "1.16.1"
	istioImageBase           = "distroless"
	defaultIstioOperatorPath = "test/test-operator.yaml"
	workingDir               = "/tmp"
)

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
			IstioVersion:   istioVersion,
			IstioImageBase: istioImageBase,
			CniEnabled:     true,
		}
		// when
		err := sidecars.Reconcile(context.TODO(), createFakeClient(&istioCr, istiod), logr.Discard())

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
			Phase: corev1.PodPhase(corev1.PodRunning),
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
