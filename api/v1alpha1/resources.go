package v1alpha1

import v1 "k8s.io/api/core/v1"

func (r *Resources) IsEqual(resources v1.ResourceRequirements) bool {
	return r.Requests.isEqual(resources.Requests) && r.Limits.isEqual(resources.Limits)
}

func (r *ResourceClaims) isEqual(resource v1.ResourceList) bool {
	return *r.Cpu == resource.Cpu().String() && *r.Memory == resource.Memory().String()
}
