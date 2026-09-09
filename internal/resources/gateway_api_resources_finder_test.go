package resources

import (
	"context"

	"github.com/kyma-project/istio/operator/pkg/labels"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("GatewayAPIResourcesFinder", func() {
	It("should return no resources when cluster is empty", func() {
		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		finder := NewGatewayAPIResourcesFinder(context.Background(), fakeClient)

		result, err := finder.FindUserCreatedGatewayAPIResources()

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("should return Gateway API resources that are not module-managed", func() {
		httpRoute := unstructured.Unstructured{}
		httpRoute.SetGroupVersionKind(gatewayAPIResourceGVKs[2]) // HTTPRoute v1
		httpRoute.SetName("my-route")
		httpRoute.SetNamespace("default")

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(&httpRoute).Build()
		finder := NewGatewayAPIResourcesFinder(context.Background(), fakeClient)

		result, err := finder.FindUserCreatedGatewayAPIResources()

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("my-route"))
		Expect(result[0].Namespace).To(Equal("default"))
	})

	It("should not return Gateway API resources that are module-managed", func() {
		httpRoute := unstructured.Unstructured{}
		httpRoute.SetGroupVersionKind(gatewayAPIResourceGVKs[2]) // HTTPRoute v1
		httpRoute.SetName("managed-route")
		httpRoute.SetNamespace("default")
		httpRoute.SetLabels(map[string]string{labels.ModuleLabelKey: labels.ModuleLabelValue})

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(&httpRoute).Build()
		finder := NewGatewayAPIResourcesFinder(context.Background(), fakeClient)

		result, err := finder.FindUserCreatedGatewayAPIResources()

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("should silently skip kinds that are not installed on the cluster", func() {
		// An empty scheme causes all List calls to fail with "no matches for kind" → silently skipped
		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()
		finder := NewGatewayAPIResourcesFinder(context.Background(), fakeClient)

		result, err := finder.FindUserCreatedGatewayAPIResources()

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("should not return GatewayClass resources created by Istio itself", func() {
		// Istio creates GatewayClass resources with controllerName "istio.io/*" during installation.
		// These must not block module deletion even though they carry no module label.
		istioGatewayClass := unstructured.Unstructured{}
		istioGatewayClass.SetGroupVersionKind(gatewayAPIResourceGVKs[0]) // GatewayClass v1
		istioGatewayClass.SetName("istio")
		Expect(unstructured.SetNestedField(istioGatewayClass.Object, "istio.io/gateway-controller", "spec", "controllerName")).To(Succeed())

		istioRemoteGatewayClass := unstructured.Unstructured{}
		istioRemoteGatewayClass.SetGroupVersionKind(gatewayAPIResourceGVKs[0])
		istioRemoteGatewayClass.SetName("istio-remote")
		Expect(unstructured.SetNestedField(istioRemoteGatewayClass.Object, "istio.io/unmanaged-gateway", "spec", "controllerName")).To(Succeed())

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).
			WithObjects(&istioGatewayClass, &istioRemoteGatewayClass).Build()
		finder := NewGatewayAPIResourcesFinder(context.Background(), fakeClient)

		result, err := finder.FindUserCreatedGatewayAPIResources()

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeEmpty())
	})

	It("should return GatewayClass resources whose controllerName is not istio.io/", func() {
		userGatewayClass := unstructured.Unstructured{}
		userGatewayClass.SetGroupVersionKind(gatewayAPIResourceGVKs[0]) // GatewayClass v1
		userGatewayClass.SetName("my-gateway-class")
		Expect(unstructured.SetNestedField(userGatewayClass.Object, "example.com/my-controller", "spec", "controllerName")).To(Succeed())

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(&userGatewayClass).Build()
		finder := NewGatewayAPIResourcesFinder(context.Background(), fakeClient)

		result, err := finder.FindUserCreatedGatewayAPIResources()

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("my-gateway-class"))
	})
})

var _ = Describe("HasAnyModuleManagedGatewayAPICRD", func() {
	It("should return false when no CRDs are present", func() {
		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()

		result := HasAnyModuleManagedGatewayAPICRD(context.Background(), fakeClient)

		Expect(result).To(BeFalse())
	})

	It("should return true when at least one module-managed CRD is present", func() {
		crd := unstructured.Unstructured{}
		crd.SetAPIVersion("apiextensions.k8s.io/v1")
		crd.SetKind("CustomResourceDefinition")
		crd.SetName("httproutes.gateway.networking.k8s.io")
		crd.SetLabels(map[string]string{labels.ModuleLabelKey: labels.ModuleLabelValue})

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(&crd).Build()

		result := HasAnyModuleManagedGatewayAPICRD(context.Background(), fakeClient)

		Expect(result).To(BeTrue())
	})

	It("should return false when CRD is present but without module label", func() {
		crd := unstructured.Unstructured{}
		crd.SetAPIVersion("apiextensions.k8s.io/v1")
		crd.SetKind("CustomResourceDefinition")
		crd.SetName("httproutes.gateway.networking.k8s.io")

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(&crd).Build()

		result := HasAnyModuleManagedGatewayAPICRD(context.Background(), fakeClient)

		Expect(result).To(BeFalse())
	})
})
