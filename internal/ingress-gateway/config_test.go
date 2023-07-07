package ingressgateway_test

import (
	"context"

	ingressgateway "github.com/kyma-project/istio/operator/internal/ingress-gateway"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	TestConfigMap string = `
accessLogEncoding: JSON
defaultConfig:
  discoveryAddress: istiod.istio-system.svc:15012
  gatewayTopology:
    numTrustedProxies: 3
  proxyMetadata: {}
  tracing:
    sampling: 100
    zipkin:
      address: zipkin.kyma-system:9411
enableTracing: true
rootNamespace: istio-system
trustDomain: cluster.local
`

	TestConfigMapEmpty string = `
accessLogEncoding: JSON
defaultConfig:
  discoveryAddress: istiod.istio-system.svc:15012
  proxyMetadata: {}
  tracing:
    sampling: 100
    zipkin:
      address: zipkin.kyma-system:9411
enableTracing: true
rootNamespace: istio-system
trustDomain: cluster.local
`
)

var _ = Describe("Config", func() {
	Context("GetNumTrustedProxyFromIstioCM", func() {
		It("should return 3 numTrustedProxies when CM has configure 3 numTrustedProxies", func() {
			//given
			client := CreateFakeClientWithIGW(TestConfigMap)

			//when
			numTrustedProxies, err := ingressgateway.GetNumTrustedProxyFromIstioCM(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(numTrustedProxies).ToNot(BeNil())
			Expect(*numTrustedProxies).To(Equal(3))
		})

		It("should return nil when CM has no configuration for numTrustedProxies", func() {
			//given
			client := CreateFakeClientWithIGW(TestConfigMapEmpty)

			//when
			numTrustedProxies, err := ingressgateway.GetNumTrustedProxyFromIstioCM(context.TODO(), client)

			//then
			Expect(err).To(Not(HaveOccurred()))
			Expect(numTrustedProxies).To(BeNil())
		})
	})
})
