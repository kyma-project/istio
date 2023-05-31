package v1alpha1_test

import (
	v1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorv1alpha1 "istio.io/api/operator/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

var _ = Describe("GetProxyResources", func() {

	It("should get resources from merged Istio CR and istio operator", func() {
		//given
		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{},
		}

		cpuRequests := "500m"
		memoryRequests := "500Mi"
		cpuLimits := "800m"
		memoryLimits := "800Mi"
		istioCR := v1alpha1.Istio{Spec: v1alpha1.IstioSpec{Components: &v1alpha1.Components{
			Proxy: &v1alpha1.ProxyComponent{K8S: &v1alpha1.ProxyK8sConfig{
				Resources: &v1alpha1.Resources{
					Requests: &v1alpha1.ResourceClaims{
						Cpu:    &cpuRequests,
						Memory: &memoryRequests,
					},
					Limits: &v1alpha1.ResourceClaims{
						Cpu:    &cpuLimits,
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
})
