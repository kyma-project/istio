package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// +kubebuilder:validation:Optional

// Config is the configuration for the Istio installation.
type Config struct {
	// Defines the number of trusted proxies deployed in front of the Istio gateway proxy.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=4294967295
	NumTrustedProxies *int `json:"numTrustedProxies,omitempty"`

	// Defines the strategy of handling the **X-Forwarded-Client-Cert** header.
	// The default behavior is "SANITIZE".
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=SANITIZE
	// +kubebuilder:validation:Enum=APPEND_FORWARD;SANITIZE_SET;SANITIZE;ALWAYS_FORWARD_ONLY;FORWARD_ONLY
	ForwardClientCertDetails *XFCCStrategy `json:"forwardClientCertDetails,omitempty"`

	// Defines a list of external authorization providers.
	Authorizers []*Authorizer `json:"authorizers,omitempty"`

	// Defines the external traffic policy for the Istio Ingress Gateway Service. Valid configurations are "Local" or "Cluster". The external traffic policy set to "Local" preserves the client IP in the request, but also introduces the risk of unbalanced traffic distribution.
	// WARNING: Switching `externalTrafficPolicy` may result in a temporal increase in request delay. Make sure that this is acceptable.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Local;Cluster
	GatewayExternalTrafficPolicy *string `json:"gatewayExternalTrafficPolicy,omitempty"`

	// Defines the telemetry configuration of Istio.
	// +kubebuilder:validation:Optional
	Telemetry Telemetry `json:"telemetry,omitempty"`

	// Defines trust domain configuration of Istio.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=`^[a-z0-9]*([a-z0-9-_]*)?(\.[a-z0-9]*([a-z0-9-_]*[a-z0-9]*)?)*$`
	TrustDomain *string `json:"trustDomain,omitempty"`
}

// Defines how to handle the x-forwarded-client-cert (XFCC) of the HTTP header.
// XFCC is a proxy header that indicates certificate information of part or all of the clients or proxies that a request has passed through on its route from the client to the server.
type XFCCStrategy string

const (
	// When the client connection is mutual TLS (mTLS), append the client certificate information to the request’s XFCC header and forward it.
	AppendForward XFCCStrategy = "APPEND_FORWARD"
	// When the client connection is mTLS, reset the XFCC header with the client certificate information and send it to the next hop.
	SanitizeSet XFCCStrategy = "SANITIZE_SET"
	// Do not send the XFCC header to the next hop.
	Sanitize XFCCStrategy = "SANITIZE"
	// Always forward the XFCC header in the request, regardless of whether the client connection is mTLS.
	AlwaysForwardOnly XFCCStrategy = "ALWAYS_FORWARD_ONLY"
	// When the client connection is mTLS, forward the XFCC header in the request.
	ForwardOnly XFCCStrategy = "FORWARD_ONLY"
)

type Components struct {
	// Pilot defines component configuration for Istiod
	Pilot *IstioComponent `json:"pilot,omitempty"`
	// IngressGateway defines component configurations for Istio Ingress Gateway
	IngressGateway *IstioComponent `json:"ingressGateway,omitempty"`
	// Cni defines component configuration for Istio CNI DaemonSet
	Cni *CniComponent `json:"cni,omitempty"`
	// Proxy defines component configuration for Istio proxy sidecar
	Proxy *ProxyComponent `json:"proxy,omitempty"`
	// +kubebuilder:validation:Optional
	EgressGateway *EgressGateway `json:"egressGateway,omitempty"`
}

// KubernetesResourcesConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec
type KubernetesResourcesConfig struct {
	// HPASpec defines configuration for HorizontalPodAutoscaler: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/
	// +kubebuilder:validation:Optional
	HPASpec *HPASpec `json:"hpaSpec,omitempty"`
	// Strategy defines configuration for rolling updates: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment
	// +kubebuilder:validation:Optional
	Strategy *Strategy `json:"strategy,omitempty"`
	// Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +kubebuilder:validation:Optional
	Resources *Resources `json:"resources,omitempty"`
}

// ProxyComponent defines configuration for Istio proxies.
type ProxyComponent struct {
	// +kubebuilder:validation:Required
	K8S *ProxyK8sConfig `json:"k8s"`
}

// ProxyK8sConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec
type ProxyK8sConfig struct {
	Resources *Resources `json:"resources,omitempty"`
}

// CniComponent defines configuration for CNI Istio component.
type CniComponent struct {
	// CniK8sConfig is a subset of https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec
	// +kubebuilder:validation:Required
	K8S *CniK8sConfig `json:"k8s"`
}

type CniK8sConfig struct {
	// Affinity defines the Pod scheduling affinity constraints: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity
	// +kubebuilder:validation:Optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +kubebuilder:validation:Optional
	Resources *Resources `json:"resources,omitempty"`
}

// HPASpec defines configuration for HorizontalPodAutoscaler.
type HPASpec struct {
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2147483647
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2147483647
	MinReplicas *int32 `json:"minReplicas,omitempty"`
}

// IstioComponent defines configuration for generic Istio component (ingress gateway, istiod).
type IstioComponent struct {
	// +kubebuilder:validation:Required
	K8s *KubernetesResourcesConfig `json:"k8s"`
}

// Strategy defines rolling update strategy.
type Strategy struct {
	// +kubebuilder:validation:Required
	RollingUpdate *RollingUpdate `json:"rollingUpdate"`
}

// RollingUpdate defines configuration for rolling updates: https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment
type RollingUpdate struct {
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Pattern=`^[0-9]+%?$`
	// +kubebuilder:validation:XValidation:rule="(type(self) == int ? self >= 0 && self <= 2147483647: self.size() >= 0)",message="must not be negative, more than 2147483647 or an empty string"
	MaxSurge *intstr.IntOrString `json:"maxSurge"       protobuf:"bytes,2,opt,name=maxSurge"`
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Pattern="^((100|[0-9]{1,2})%|[0-9]+)$"
	// +kubebuilder:validation:XValidation:rule="(type(self) == int ? self >= 0 && self <= 2147483647: self.size() >= 0)",message="must not be negative, more than 2147483647 or an empty string"
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable" protobuf:"bytes,1,opt,name=maxUnavailable"`
}

// Resources define Kubernetes resources configuration: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
type Resources struct {
	Limits   *ResourceClaims `json:"limits,omitempty"`
	Requests *ResourceClaims `json:"requests,omitempty"`
}

type ResourceClaims struct {
	// +kubebuilder:validation:Pattern=`^([0-9]+m?|[0-9]\.[0-9]{1,3})$`
	CPU *string `json:"cpu,omitempty"`

	// +kubebuilder:validation:Pattern=`^[0-9]+(((\.[0-9]+)?(E|P|T|G|M|k|Ei|Pi|Ti|Gi|Mi|Ki|m)?)|(e[0-9]+))$`
	Memory *string `json:"memory,omitempty"`
}

// EgressGateway defines configuration for Istio egressGateway.
type EgressGateway struct {
	// Defines the Kubernetes resources configuration for Istio egress gateway.
	// +kubebuilder:validation:Optional
	K8s *KubernetesResourcesConfig `json:"k8s"`
	// Enables or disables the Istio egress gateway.
	// +kubebuilder:validation:Optional
	Enabled *bool `json:"enabled,omitempty"`
}
