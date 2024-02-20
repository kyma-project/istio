// +kubebuilder:validation:Optional
package v1alpha2

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Config is the configuration for the Istio installation.
type Config struct {
	// Defines the number of trusted proxies deployed in front of the Istio gateway proxy.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=4294967295
	NumTrustedProxies *int `json:"numTrustedProxies,omitempty"`

	// Defines a list of external authorization providers.
	Authorizers []*Authorizer `json:"authorizers,omitempty"`
}

type Components struct {
	// Pilot defines component configuration for Istiod
	Pilot *IstioComponent `json:"pilot,omitempty"`
	// IngressGateway defines component configurations for Istio Ingress Gateway
	IngressGateway *IstioComponent `json:"ingressGateway,omitempty"`
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
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2147483647
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2147483647
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
	// +kubebuilder:validation:Pattern=`^[0-9]+%?$`
	// +kubebuilder:validation:XValidation:rule="(type(self) == int ? self >= 0 && self <= 2147483647: self.size() >= 0)",message="must not be negative, more than 2147483647 or an empty string"
	MaxSurge *intstr.IntOrString `json:"maxSurge" protobuf:"bytes,2,opt,name=maxSurge"`
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
	Cpu *string `json:"cpu,omitempty"`

	// +kubebuilder:validation:Pattern=`^[0-9]+(((\.[0-9]+)?(E|P|T|G|M|k|Ei|Pi|Ti|Gi|Mi|Ki|m)?)|(e[0-9]+))$`
	Memory *string `json:"memory,omitempty"`
}

type Authorizer struct {
	// A unique name identifying the extension authorization provider.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Specifies the service that implements the Envoy ext_authz HTTP authorization service.
	// The format is "[<Namespace>/]<Hostname>".
	// The specification of "<Namespace>"
	// is required only when it is insufficient to unambiguously resolve a service in the service registry.
	// The "<Hostname>" is a fully qualified host name of a service defined by the Kubernetes service or ServiceEntry.
	// The recommended format is "[<Namespace>/]<Hostname>"
	// Example: "my-ext-authz.foo.svc.cluster.local" or "bar/my-ext-authz".
	// +kubebuilder:validation:Required
	Service string `json:"service"`

	// Specifies the port of the service.
	// +kubebuilder:validation:Required
	Port uint32 `json:"port"`

	// Specifies headers to be included, added or forwarded during authorization.
	Headers *Headers `json:"headers,omitempty"`
}

// Exact, prefix and suffix matches are supported (similar to the authorization policy rule syntax except the presence match
// https://istio.io/latest/docs/reference/config/security/authorization-policy/#Rule):
// - Exact match: "abc" will match on value "abc".
// - Prefix match: "abc*" will match on value "abc" and "abcd".
// - Suffix match: "*abc" will match on value "abc" and "xabc".

type Headers struct {
	// Defines headers to be included or added in check authorization request.
	InCheck *InCheck `json:"inCheck,omitempty"`

	// Defines headers to be forwarded to the upstream.
	ToUpstream *ToUpstream `json:"toUpstream,omitempty"`

	// Defines headers to be forwarded to the downstream.
	ToDownstream *ToDownstream `json:"toDownstream,omitempty"`
}

type InCheck struct {
	// List of client request headers that should be included in the authorization request sent to the authorization service.
	// Note that in addition to the headers specified here, the following headers are included by default:
	// 1. *Host*, *Method*, *Path* and *Content-Length* are automatically sent.
	// 2. *Content-Length* will be set to 0, and the request will not have a message body. However, the authorization request can include the buffered client request body (controlled by include_request_body_in_check setting), consequently the value of Content-Length of the authorization request reflects the size of its payload size.
	Include []string `json:"include,omitempty"`

	// Set of additional fixed headers that should be included in the authorization request sent to the authorization service.
	// The Key is the header name and value is the header value.
	// Note that client request of the same key or headers specified in `Include` will be overridden.
	Add map[string]string `json:"add,omitempty"`
}

type ToUpstream struct {
	// List of headers from the authorization service that should be added or overridden in the original request and forwarded to the upstream when the authorization check result is allowed (HTTP code 200).
	// If not specified, the original request will not be modified and forwarded to backend as-is.
	// Note, any existing headers will be overridden.
	OnAllow []string `json:"onAllow,omitempty"`
}

type ToDownstream struct {
	// List of headers from the authorization service that should be forwarded to downstream when the authorization check result is allowed (HTTP code 200).
	// If not specified, the original response will not be modified and forwarded to downstream as-is.
	// Note, any existing headers will be overridden.
	OnAllow []string `json:"onAllow,omitempty"`

	// List of headers from the authorization service that should be forwarded to downstream when the authorization check result is not allowed (HTTP code other than 200).
	// If not specified, all the authorization response headers, except *Authority (Host)* will be in the response to the downstream.
	// When a header is included in this list, *Path*, *Status*, *Content-Length*, *WWWAuthenticate* and *Location* are automatically added.
	// Note, the body from the authorization service is always included in the response to downstream.
	OnDeny []string `json:"onDeny,omitempty"`
}
