package clusterconfig_test

import (
	"context"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Hyperscaler", func() {
	Context("IsHyperscalerAWS", func() {
		It("should be true if hyperscaler is aws", func() {
			// given
			cm := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: clusterconfig.ConfigMapShootInfoName, Namespace: clusterconfig.ConfigMapShootInfoNS},
				Data:       map[string]string{"provider": "aws"},
			}

			k8sClient := createFakeClient(&cm)
			// when
			isAws, err := clusterconfig.IsHyperscalerAWS(context.Background(), k8sClient)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(isAws).To(BeTrue())
		})

		It("should be false if hyperscaler is not aws", func() {
			// given
			cm := v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{Name: clusterconfig.ConfigMapShootInfoName, Namespace: clusterconfig.ConfigMapShootInfoNS},
				Data:       map[string]string{"provider": "gcp"},
			}

			k8sClient := createFakeClient(&cm)

			// when
			isAws, err := clusterconfig.IsHyperscalerAWS(context.Background(), k8sClient)

			// then
			Expect(err).ToNot(HaveOccurred())
			Expect(isAws).To(BeFalse())
		})

	})
})
