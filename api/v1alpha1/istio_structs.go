// +kubebuilder:validation:Optional
package v1alpha1

import "k8s.io/apimachinery/pkg/util/intstr"

// Config is the configuration for the Istio installation.
type Config struct {
	// Defines the number of trusted proxies deployed in front of the Istio gateway proxy.
	NumTrustedProxies *int `json:"numTrustedProxies,omitempty"`
}

type Components struct {
	Pilot           *IstioComponent   `json:"pilot,omitempty"`
	IngressGateways []*IstioComponent `json:"ingressGateways,omitempty"`
}

// KubernetesResourcesConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec
type KubernetesResourcesConfig struct {
	HPASpec   *HPASpec   `json:"hpaSpec,omitempty"`
	Strategy  *Strategy  `json:"strategy,omitempty"`
	Resources *Resources `json:"resources,omitempty"`
}

type HPASpec struct {
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	MinReplicas *int32 `json:"minReplicas,omitempty"`
}

type IstioComponent struct {
	// +kubebuilder:validation:Required
	K8s KubernetesResourcesConfig `json:"k8s"`
}

type Strategy struct {
	// +kubebuilder:validation:Required
	RollingUpdate RollingUpdate `json:"rollingUpdate"`
}

type RollingUpdate struct {
	// +kubebuilder:validation:XIntOrString
	MaxSurge *intstr.IntOrString `json:"maxSurge" protobuf:"bytes,2,opt,name=maxSurge"`
	// +kubebuilder:validation:XIntOrString
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable" protobuf:"bytes,1,opt,name=maxUnavailable"`
}

type Resources struct {
	Limits   *ResourceClaims `json:"limits,omitempty"`
	Requests *ResourceClaims `json:"requests,omitempty"`
}

type ResourceClaims struct {
	Cpu    *string `json:"cpu,omitempty"`
	Memory *string `json:"memory,omitempty"`
}
