// +kubebuilder:validation:Optional
package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Config is the configuration for the Istio installation.
type Config struct {
	// Defines the number of trusted proxies deployed in front of the Istio gateway proxy.
	NumTrustedProxies *int `json:"numTrustedProxies,omitempty"`
}

type Components struct {
	// Pilot defines component configuration for Istiod
	Pilot *IstioComponent `json:"pilot,omitempty"`
	// IngressGateways defines component configurations for Istio Ingress Gateways
	IngressGateways *IstioComponent `json:"ingressGateways,omitempty"`
	// Cni defines component configuration for Istio CNI DaemonSet
	Cni *CniComponent `json:"cni,omitempty"`
	// Proxy defines component configuration for Istio proxy sidecar
	Proxy *ProxyComponent `json:"proxy,omitempty"`
}

// KubernetesResourcesConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec
type KubernetesResourcesConfig struct {
	HPASpec   *HPASpec   `json:"hpaSpec,omitempty"`
	Strategy  *Strategy  `json:"strategy,omitempty"`
	Resources *Resources `json:"resources,omitempty"`
}

// ProxyComponent defines configuration for Istio proxies
type ProxyComponent struct {
	// +kubebuilder:validation:Required
	K8S *ProxyK8sConfig `json:"k8s"`
}

// ProxyK8sConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec
type ProxyK8sConfig struct {
	Resources *Resources `json:"resources,omitempty"`
}

// CniComponent defines configuration for CNI Istio component
type CniComponent struct {
	// +kubebuilder:validation:Required
	K8S *CniK8sConfig `json:"k8s"`
}

// CniK8sConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec
type CniK8sConfig struct {
	Affinity  *v1.Affinity `json:"affinity,omitempty"`
	Resources *Resources   `json:"resources,omitempty"`
}

// HPASpec defines configuration for HorizontalPodAutoscaler
type HPASpec struct {
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`
	MinReplicas *int32 `json:"minReplicas,omitempty"`
}

// IstioComponent defines configuration for generic Istio component (ingress gateway, istiod)
type IstioComponent struct {
	// +kubebuilder:validation:Required
	K8s *KubernetesResourcesConfig `json:"k8s"`
}

// Strategy defines rolling update strategy
type Strategy struct {
	// +kubebuilder:validation:Required
	RollingUpdate *RollingUpdate `json:"rollingUpdate"`
}

// RollingUpdate defines configuration for rolling updates: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment
type RollingUpdate struct {
	// +kubebuilder:validation:XIntOrString
	MaxSurge *intstr.IntOrString `json:"maxSurge" protobuf:"bytes,2,opt,name=maxSurge"`
	// +kubebuilder:validation:XIntOrString
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable" protobuf:"bytes,1,opt,name=maxUnavailable"`
}

// Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
type Resources struct {
	Limits   *ResourceClaims `json:"limits,omitempty"`
	Requests *ResourceClaims `json:"requests,omitempty"`
}

type ResourceClaims struct {
	Cpu    *string `json:"cpu,omitempty"`
	Memory *string `json:"memory,omitempty"`
}
