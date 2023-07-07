package ingressgateway_test

import (
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	namespace      string = "istio-system"
	deploymentName string = "istio-ingressgateway"
)

func CreateFakeClientWithIGW(configMaps ...string) client.Client {
	igwDeployment := appsv1.Deployment{ObjectMeta: v1.ObjectMeta{Name: deploymentName, Namespace: namespace}}

	err := corev1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).ShouldNot(HaveOccurred())

	if len(configMaps) > 0 {
		data := make(map[string]string)
		data["mesh"] = configMaps[0]
		return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(&corev1.ConfigMap{ObjectMeta: v1.ObjectMeta{Name: "istio", Namespace: namespace}, Data: data}, &igwDeployment).Build()
	}
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(&igwDeployment).Build()
}
