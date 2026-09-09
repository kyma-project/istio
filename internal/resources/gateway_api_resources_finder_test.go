package resources

import (
	"context"
	"fmt"

	"github.com/kyma-project/istio/operator/pkg/labels"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

	// Regression test: ListenerSet must be queried as v1, matching the version served by the embedded CRD.
	// A version mismatch would cause the List call to be skipped silently ("no matches for kind"),
	// so user-created ListenerSets would not block uninstall.
	It("should return a user-created ListenerSet (v1) as a blocking resource", func() {
		listenerSet := unstructured.Unstructured{}
		listenerSet.SetGroupVersionKind(gatewayAPIResourceGVKs[6]) // ListenerSet v1
		listenerSet.SetName("my-listener-set")
		listenerSet.SetNamespace("default")

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(&listenerSet).Build()
		finder := NewGatewayAPIResourcesFinder(context.Background(), fakeClient)

		result, err := finder.FindUserCreatedGatewayAPIResources()

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(1))
		Expect(result[0].Name).To(Equal("my-listener-set"))
		Expect(result[0].GVK.Version).To(Equal("v1"))
	})

	// Drift guard: every GVK in gatewayAPIResourceGVKs must be exercised so that a version mismatch
	// in the GVK table (like the v1alpha2 vs v1 issue for ListenerSet) is caught by tests.
	It("should find resources for every registered GVK", func() {
		objects := make([]client.Object, len(gatewayAPIResourceGVKs))
		for i, gvk := range gatewayAPIResourceGVKs {
			obj := &unstructured.Unstructured{}
			obj.SetGroupVersionKind(gvk)
			obj.SetName(fmt.Sprintf("resource-%d", i))
			if gvk.Kind != "GatewayClass" {
				obj.SetNamespace("default")
			}
			objects[i] = obj
		}

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(objects...).Build()
		finder := NewGatewayAPIResourcesFinder(context.Background(), fakeClient)

		result, err := finder.FindUserCreatedGatewayAPIResources()

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(HaveLen(len(gatewayAPIResourceGVKs)),
			"every GVK in gatewayAPIResourceGVKs must be reachable; a mismatch causes silent skipping")
	})
})

var _ = Describe("HasAnyModuleManagedGatewayAPICRD", func() {
	It("should return false when no CRDs are present", func() {
		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).Build()

		result, err := HasAnyModuleManagedGatewayAPICRD(context.Background(), fakeClient)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("should return true when at least one module-managed CRD is present", func() {
		crd := unstructured.Unstructured{}
		crd.SetAPIVersion("apiextensions.k8s.io/v1")
		crd.SetKind("CustomResourceDefinition")
		crd.SetName("httproutes.gateway.networking.k8s.io")
		crd.SetLabels(map[string]string{labels.ModuleLabelKey: labels.ModuleLabelValue})

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(&crd).Build()

		result, err := HasAnyModuleManagedGatewayAPICRD(context.Background(), fakeClient)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeTrue())
	})

	It("should return false when CRD is present but without module label", func() {
		crd := unstructured.Unstructured{}
		crd.SetAPIVersion("apiextensions.k8s.io/v1")
		crd.SetKind("CustomResourceDefinition")
		crd.SetName("httproutes.gateway.networking.k8s.io")

		fakeClient := fake.NewClientBuilder().WithScheme(runtime.NewScheme()).WithObjects(&crd).Build()

		result, err := HasAnyModuleManagedGatewayAPICRD(context.Background(), fakeClient)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(BeFalse())
	})

	It("should propagate a non-NotFound error rather than silently returning false", func() {
		// A client that returns a generic error for any Get call simulates an auth or transport failure.
		// The safety check must not be bypassed in this case.
		fakeClient := &errorInjectingClient{err: fmt.Errorf("forbidden")}

		result, err := HasAnyModuleManagedGatewayAPICRD(context.Background(), fakeClient)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("forbidden"))
		Expect(result).To(BeFalse())
	})
})

// errorInjectingClient is a minimal client.Client stub that returns a fixed error for every Get call.
type errorInjectingClient struct {
	client.Client
	err error
}

func (c *errorInjectingClient) Get(_ context.Context, _ client.ObjectKey, _ client.Object, _ ...client.GetOption) error {
	return c.err
}
