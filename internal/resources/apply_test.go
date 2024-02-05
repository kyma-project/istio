package resources_test

import (
	"context"
	_ "embed"

	"github.com/kyma-project/istio/operator/internal/resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1beta12 "istio.io/api/security/v1beta1"
	"istio.io/client-go/pkg/apis/security/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"
)

//go:embed test_files/resource_with_spec.yaml
var resourceWithSpec []byte

//go:embed test_files/resource_with_data.yaml
var resourceWithData []byte

var _ = Describe("Apply", func() {
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
		um, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&pa)
		unstr := unstructured.Unstructured{Object: um}
		Expect(err).ToNot(HaveOccurred())
		ok := resources.HasManagedByDisclaimer(unstr)
		Expect(ok).To(BeTrue())
	})

	It("should create resource containing app.kubernetes.io/version label", func() {
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
		um, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&pa)
		unstr := unstructured.Unstructured{Object: um}
		Expect(err).ToNot(HaveOccurred())

		Expect(unstr.GetLabels()).ToNot(BeNil())
		Expect(unstr.GetLabels()).To(HaveLen(1))
		Expect(unstr.GetLabels()).To(HaveKeyWithValue("app.kubernetes.io/version", "dev"))
	})

	It("should update resource with spec and add disclaimer", func() {
		// given
		var pa v1beta1.PeerAuthentication
		Expect(yaml.Unmarshal(resourceWithSpec, &pa)).Should(Succeed())
		k8sClient := createFakeClient(&pa)

		pa.Spec.Mtls.Mode = v1beta12.PeerAuthentication_MutualTLS_PERMISSIVE
		var resourceWithUpdatedSpec []byte
		resourceWithUpdatedSpec, err := yaml.Marshal(&pa)
		Expect(err).ShouldNot(HaveOccurred())

		// when
		res, err := resources.Apply(context.Background(), k8sClient, resourceWithUpdatedSpec, nil)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(controllerutil.OperationResultUpdated))

		Expect(k8sClient.Get(context.Background(), ctrlClient.ObjectKeyFromObject(&pa), &pa)).Should(Succeed())
		Expect(pa.Spec.Mtls.Mode).To(Equal(v1beta12.PeerAuthentication_MutualTLS_PERMISSIVE))
		um, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&pa)
		unstr := unstructured.Unstructured{Object: um}
		Expect(err).ToNot(HaveOccurred())
		ok := resources.HasManagedByDisclaimer(unstr)
		Expect(ok).To(BeTrue())
	})

	It("should update data field of resource and add disclaimer", func() {
		// given
		var cm v1.ConfigMap
		Expect(yaml.Unmarshal(resourceWithData, &cm)).Should(Succeed())
		k8sClient := createFakeClient(&cm)

		cm.Data["some"] = "new-data"
		var resourceWithUpdatedData []byte
		resourceWithUpdatedData, err := yaml.Marshal(&cm)

		Expect(err).ShouldNot(HaveOccurred())

		// when
		res, err := resources.Apply(context.Background(), k8sClient, resourceWithUpdatedData, nil)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(controllerutil.OperationResultUpdated))

		Expect(k8sClient.Get(context.Background(), ctrlClient.ObjectKeyFromObject(&cm), &cm)).Should(Succeed())
		Expect(cm.Data["some"]).To(Equal("new-data"))

		um, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cm)
		unstr := unstructured.Unstructured{Object: um}
		Expect(err).ToNot(HaveOccurred())
		ok := resources.HasManagedByDisclaimer(unstr)
		Expect(ok).To(BeTrue())
	})

	It("should set owner reference of resource when owner reference is given", func() {
		// given
		k8sClient := createFakeClient()
		ownerReference := metav1.OwnerReference{
			APIVersion: "security.istio.io/v1beta1",
			Kind:       "PeerAuthentication",
			Name:       "owner-name",
			UID:        "owner-uid",
		}
		// when
		res, err := resources.Apply(context.Background(), k8sClient, resourceWithSpec, &ownerReference)

		// then
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(Equal(controllerutil.OperationResultCreated))

		var pa v1beta1.PeerAuthentication
		Expect(yaml.Unmarshal(resourceWithSpec, &pa)).Should(Succeed())
		Expect(k8sClient.Get(context.Background(), ctrlClient.ObjectKeyFromObject(&pa), &pa)).Should(Succeed())
		Expect(pa.OwnerReferences).To(ContainElement(ownerReference))
	})

})
