package v1alpha2_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo tests are generally written without a direct package reference
	. "github.com/onsi/gomega"    //nolint:revive // Gomega asserts are generally written without a direct package reference
	iopv1alpha1 "istio.io/istio/operator/pkg/apis"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
)

var _ = Describe("GetProxyResources", func() {
	It("should get resources from merged Istio CR and istio operator", func() {
		// given
		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{},
		}

		cpuRequests := "500m"
		memoryRequests := "500Mi"
		cpuLimits := "800m"
		memoryLimits := "800Mi"
		istioCR := v1alpha2.Istio{Spec: v1alpha2.IstioSpec{Components: &v1alpha2.Components{
			Proxy: &v1alpha2.ProxyComponent{K8S: &v1alpha2.ProxyK8sConfig{
				Resources: &v1alpha2.Resources{
					Requests: &v1alpha2.ResourceClaims{
						CPU:    &cpuRequests,
						Memory: &memoryRequests,
					},
					Limits: &v1alpha2.ResourceClaims{
						CPU:    &cpuLimits,
						Memory: &memoryLimits,
					},
				},
			}},
		}}}

		// when
		result, err := istioCR.GetProxyResources(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		Expect(result.Requests.Cpu().String()).To(Equal(cpuRequests))
		Expect(result.Requests.Memory().String()).To(Equal(memoryRequests))
		Expect(result.Limits.Cpu().String()).To(Equal(cpuLimits))
		Expect(result.Limits.Memory().String()).To(Equal(memoryLimits))
	})

	It("should validate that resources can be returned", func() {
		// given
		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{},
		}

		istioCR := v1alpha2.Istio{Spec: v1alpha2.IstioSpec{Components: &v1alpha2.Components{
			Proxy: &v1alpha2.ProxyComponent{K8S: &v1alpha2.ProxyK8sConfig{
				Resources: &v1alpha2.Resources{
					Requests: &v1alpha2.ResourceClaims{
						CPU:    ptr.To(string("500m")),
						Memory: ptr.To(string("500Mi")),
					},
				},
			}},
		}}}

		// when
		_, err := istioCR.GetProxyResources(iop)

		// then
		Expect(err).Should(HaveOccurred())
		Expect(err.Error()).To(Equal("proxy resources missing in merged IstioOperator"))
	})

	It("should be able to get resources from real istio operator template when IstioCR has no overrides", func() {
		// given
		istioOperator, err := os.ReadFile("../../internal/istiooperator/istio-operator.yaml")
		Expect(err).ShouldNot(HaveOccurred())

		iop := iopv1alpha1.IstioOperator{}
		err = yaml.Unmarshal(istioOperator, &iop)
		Expect(err).ShouldNot(HaveOccurred())

		istioCR := v1alpha2.Istio{}

		// when
		result, err := istioCR.GetProxyResources(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		Expect(result.Requests.Cpu().String()).ToNot(BeEmpty())
		Expect(result.Requests.Memory().String()).ToNot(BeEmpty())
		Expect(result.Limits.Cpu().String()).ToNot(BeEmpty())
		Expect(result.Limits.Memory().String()).ToNot(BeEmpty())
	})
})
