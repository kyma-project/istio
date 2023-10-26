package resources_test

import (
	"context"
	_ "embed"
	"github.com/kyma-project/istio/operator/internal/resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"istio.io/client-go/pkg/apis/security/v1beta1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed test_files/resource_with_spec.yaml
var resourceWithSpec []byte

var _ = Describe("Apply resource", func() {
	It("should create resource with disclaimer", func() {
		// given
		k8sClient := createFakeClient()

		// when
		res, err := resources.Apply(context.Background(), k8sClient, resourceWithSpec, nil)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(controllerutil.OperationResultCreated))

		var pa v1beta1.PeerAuthentication
		Expect(yaml.Unmarshal(resourceWithSpec, &pa)).Should(Succeed())
		Expect(k8sClient.Get(context.Background(), ctrlClient.ObjectKeyFromObject(&pa), &pa)).Should(Succeed())
		_, ok := pa.GetAnnotations()[resources.DisclaimerKey]
		Expect(ok).To(BeTrue())

	})
	It("should update resource with disclaimer", func() {})
	It("should set owner reference of resource when owner reference is given", func() {})
	It("should update spec field of resource", func() {})
	It("should update data field of resource", func() {})

})
