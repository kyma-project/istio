package v1alpha2

import (
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GetProxyResources returns the proxy resources by merging the given IstioOperator with the configuration in Istio CR.
func (i *Istio) GetProxyResources(op istioOperator.IstioOperator) (v1.ResourceRequirements, error) {
	mergedOp, err := i.MergeInto(op)
	if err != nil {
		return v1.ResourceRequirements{}, err
	}

	if !hasResources(mergedOp) {
		return v1.ResourceRequirements{}, errors.New("proxy resources missing in merged IstioOperator")
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

func hasResources(op istioOperator.IstioOperator) bool {
	if op.Spec.Values.Fields[globalField] == nil {
		return false
	}

	if op.Spec.Values.Fields[globalField].GetStructValue().Fields[proxyField] == nil {
		return false
	}

	if op.Spec.Values.Fields[globalField].GetStructValue().Fields[proxyField].GetStructValue().Fields[resourcesField] == nil {
		return false
	}

	resources := op.Spec.Values.Fields[globalField].GetStructValue().Fields[proxyField].GetStructValue().Fields[resourcesField].GetStructValue()
	if resources.Fields[requestsField] == nil || hasNoCpuAndMemory(resources.Fields[requestsField]) {
		return false
	}
	if resources.Fields[limitsField] == nil || hasNoCpuAndMemory(resources.Fields[limitsField]) {
		return false
	}

	return true
}

func hasNoCpuAndMemory(v *structpb.Value) bool {
	return v.GetStructValue().Fields[cpu] == nil ||
		v.GetStructValue().Fields[memory] == nil
}
