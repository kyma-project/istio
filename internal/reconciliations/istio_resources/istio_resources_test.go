package istio_resources_test

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Istio resources", func() {
	Context("Get", func() {
		It("should contain proxy-protocol EnvoyFilter when hyperscaler is AWS", func() {
			// given
			cm := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: clusterconfig.ConfigMapShootInfoName, Namespace: clusterconfig.ConfigMapShootInfoNS},
				Data:       map[string]string{"provider": "aws"},
			}

			k8sClient := fake.NewClientBuilder().WithObjects(&cm).Build()

			// when
			istioResources, err := istio_resources.Get(context.Background(), k8sClient)

			// then
			Expect(err).ToNot(HaveOccurred())

			found := false
			for _, ir := range istioResources {
				if ir.Name() == "EnvoyFilter/proxy-protocol" {
					found = true
				}
			}
			Expect(found).To(BeTrue())
		})
		It("should not contain proxy-protocol EnvoyFilter when hyperscaler is not AWS", func() {
			// given
			cm := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: clusterconfig.ConfigMapShootInfoName, Namespace: clusterconfig.ConfigMapShootInfoNS},
				Data:       map[string]string{"provider": "gcp"},
			}

			k8sClient := fake.NewClientBuilder().WithObjects(&cm).Build()

			// when
			istioResources, err := istio_resources.Get(context.Background(), k8sClient)

			// then
			Expect(err).ToNot(HaveOccurred())
			for _, ir := range istioResources {
				if ir.Name() == "EnvoyFilter/proxy-protocol" {
					Fail("Should not find EnvoyFilter/proxy-protocol")
				}
			}
		})
	})
})
