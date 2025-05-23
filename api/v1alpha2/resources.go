package v1alpha2

import (
	"github.com/pkg/errors"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis"
	"istio.io/istio/operator/pkg/values"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// GetProxyResources returns the proxy resources by merging the given IstioOperator with the configuration in Istio CR.
func (i *Istio) GetProxyResources(op iopv1alpha1.IstioOperator) (v1.ResourceRequirements, error) {
	mergedOp, err := i.MergeInto(op)
	if err != nil {
		return v1.ResourceRequirements{}, err
	}

	if !hasResources(mergedOp) {
		return v1.ResourceRequirements{}, errors.New("proxy resources missing in merged IstioOperator")
	}

	valuesMap, err := values.MapFromObject(mergedOp.Spec.Values)
	if err != nil {
		return v1.ResourceRequirements{}, err
	}

	resources, exists := valuesMap.GetPathMap("global.proxy.resources")
	if !exists {
		return v1.ResourceRequirements{}, err
	}

	cpuRequest := resources.GetPathString("requests.cpu")
	memoryRequest := resources.GetPathString("requests.memory")
	cpuLimit := resources.GetPathString("limits.cpu")
	memoryLimit := resources.GetPathString("limits.memory")

	//nolint:exhaustive // no need to set all values for resource requirements
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

func hasResources(op iopv1alpha1.IstioOperator) bool {
	valuesMap, err := values.MapFromObject(op.Spec.Values)
	if err != nil {
		return false
	}
	resourcesMap, exists := valuesMap.GetPathMap("global.proxy.resources")
	if !exists {
		return false
	}

	requests, exists := resourcesMap.GetPathMap("requests")
	if !exists {
		return false
	}
	limits, exists := resourcesMap.GetPathMap("limits")
	if !exists {
		return false
	}
	if hasNoCPUAndMemory(requests) {
		return false
	}
	if hasNoCPUAndMemory(limits) {
		return false
	}

	return true
}

func hasNoCPUAndMemory(m values.Map) bool {
	return m.GetPathString("cpu") == "" || m.GetPathString("memory") == ""
}
