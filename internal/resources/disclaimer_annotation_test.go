package resources

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	client2 "sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Create resources", func() {
	It("Should annotate with disclaimer when there was no such annotation", func() {
		client := createFakeClient()
		unstr := unstructured.Unstructured{Object: map[string]interface{}{}}
		unstr.SetName("test")
		err := AnnotateWithDisclaimer(context.Background(), unstr, client)
		Expect(err).ToNot(HaveOccurred())

		_ = client.Get(context.Background(), client2.ObjectKey{Name: unstr.GetName()}, &unstr)
		anns := unstr.GetAnnotations()
		Expect(anns[DisclaimerKey]).To(Equal())

	})
	It("Should not change existing disclaimer annotation", func() {
		client := createFakeClient()
	})
})
