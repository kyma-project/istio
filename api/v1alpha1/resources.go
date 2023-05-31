package v1alpha1

import (
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (i *Istio) GetProxyResources(op istioOperator.IstioOperator) (v1.ResourceRequirements, error) {
	mergedOp, err := i.MergeInto(op)
	if err != nil {
		return v1.ResourceRequirements{}, err
	}
	resources := mergedOp.Spec.Values.
		Fields[globalField].GetStructValue().
		Fields[proxyField].GetStructValue().
		Fields[resourcesField].GetStructValue()

	cpuRequest := resources.Fields[requestsField].GetStructValue().Fields[cpu].GetStringValue()
	memoryRequest := resources.Fields[requestsField].GetStructValue().Fields[memory].GetStringValue()
	cpuLimit := resources.Fields[limitsField].GetStructValue().Fields[cpu].GetStringValue()
	memoryLimit := resources.Fields[limitsField].GetStructValue().Fields[memory].GetStringValue()

	return v1.ResourceRequirements{
		Requests: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(cpuRequest),
			v1.ResourceMemory: resource.MustParse(memoryRequest),
		},
		Limits: v1.ResourceList{
			v1.ResourceCPU:    resource.MustParse(cpuLimit),
			v1.ResourceMemory: resource.MustParse(memoryLimit),
		},
	}, nil
}
