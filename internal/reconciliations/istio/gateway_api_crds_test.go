package istio_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
)

// TODO(gateway-api-parked): This feature was parked in April 2026.
// This test file needs full rewrite of test cases based on new gateway_api_crds.go implementation.

var _ = Describe("Gateway API CRDs", func() {
	var (
		ctx                    context.Context
		gatewayAPICRDInstaller *istio.GatewayAPICRDInstaller
		fakeClient             client.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		// Register CRD scheme
		_ = apiextensionsv1.AddToScheme(scheme.Scheme)
		fakeClient = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		gatewayAPICRDInstaller = istio.NewGatewayAPICRDInstaller(fakeClient)
	})

	Describe("Install", func() {
		It("should install Gateway API CRDs successfully", func() {
			// When
			err := gatewayAPICRDInstaller.Install(ctx)

			// Then
			Expect(err).ToNot(HaveOccurred())

			// Verify key Gateway API CRDs are installed
			gatewayCRD := &apiextensionsv1.CustomResourceDefinition{}
			err = fakeClient.Get(ctx, client.ObjectKey{Name: "gateways.gateway.networking.k8s.io"}, gatewayCRD)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should be idempotent when called multiple times", func() {
			// When - Install twice
			err := gatewayAPICRDInstaller.Install(ctx)
			Expect(err).ToNot(HaveOccurred())

			err = gatewayAPICRDInstaller.Install(ctx)

			// Then - Should succeed without error
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("IsInstalled", func() {
		It("should return true when Gateway API CRDs are installed", func() {
			// Given
			err := gatewayAPICRDInstaller.Install(ctx)
			Expect(err).ToNot(HaveOccurred())

			// When
			installed, err := gatewayAPICRDInstaller.IsInstalled(ctx)

			// Then
			Expect(err).ToNot(HaveOccurred())
			Expect(installed).To(BeTrue())
		})

		It("should return false when Gateway API CRDs are not installed", func() {
			// When
			installed, err := gatewayAPICRDInstaller.IsInstalled(ctx)

			// Then
			Expect(err).ToNot(HaveOccurred())
			Expect(installed).To(BeFalse())
		})
	})

	Describe("Uninstall", func() {
		It("should remove Gateway API CRDs successfully", func() {
			// Given - Install first
			err := gatewayAPICRDInstaller.Install(ctx)
			Expect(err).ToNot(HaveOccurred())

			// When
			err = gatewayAPICRDInstaller.Uninstall(ctx)

			// Then
			Expect(err).ToNot(HaveOccurred())

			// Verify CRDs are removed
			gatewayCRD := &apiextensionsv1.CustomResourceDefinition{}
			err = fakeClient.Get(ctx, client.ObjectKey{Name: "gateways.gateway.networking.k8s.io"}, gatewayCRD)
			Expect(apierrors.IsNotFound(err)).To(BeTrue())
		})

		It("should not fail when CRDs are already removed", func() {
			// When - Try to uninstall when nothing is installed
			err := gatewayAPICRDInstaller.Uninstall(ctx)

			// Then - Should succeed without error
			Expect(err).ToNot(HaveOccurred())
		})
	})
})
