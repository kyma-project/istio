package v1alpha2

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Configures the Istio installation.
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

	// Defines the external traffic policy for the Istio Ingress Gateway Service. Valid configurations are `"Local"` or `"Cluster"`. The external traffic policy set to `"Local"` preserves the client IP in the request, but also introduces the risk of unbalanced traffic distribution.
	// WARNING: Switching **externalTrafficPolicy** may result in a temporal increase in request delay. Make sure that this is acceptable.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=Local;Cluster
	GatewayExternalTrafficPolicy *string `json:"gatewayExternalTrafficPolicy,omitempty"`

	// Defines the telemetry configuration of Istio.
	// +kubebuilder:validation:Optional
	Telemetry Telemetry `json:"telemetry,omitempty"`
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
	// Configures the Istiod component.
	Pilot *IstioComponent `json:"pilot,omitempty"`
	// Configures the Istio Ingress Gateway component.
	IngressGateway *IstioComponent `json:"ingressGateway,omitempty"`
	// Configures the Istio CNI DaemonSet component.
	Cni *CniComponent `json:"cni,omitempty"`
	// Configures the Istio sidecar proxy component.
	Proxy *ProxyComponent `json:"proxy,omitempty"`
	// Configures the Istio Egress Gateway component.
	// +kubebuilder:validation:Optional
	EgressGateway *EgressGateway `json:"egressGateway,omitempty"`
}

// Defines Kubernetes-level configuration options for Istio components. It's a subset of [KubernetesResourcesSpec](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec).
type KubernetesResourcesConfig struct {
	// Configures the [HorizontalPodAutoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).
	// +kubebuilder:validation:Optional
	HPASpec *HPASpec `json:"hpaSpec,omitempty"`
	// Defines the rolling updates strategy. See [Rolling Update Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment).
	// +kubebuilder:validation:Optional
	Strategy *Strategy `json:"strategy,omitempty"`
	// Defines Kubernetes resources' configuration. See [Resource Management for Pods and Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).
	// +kubebuilder:validation:Optional
	Resources *Resources `json:"resources,omitempty"`
}

// Configures the Istio sidecar proxy component.
type ProxyComponent struct {
	// **ProxyK8sConfig** is a subset of [KubernetesResourcesSpec](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec).
	// +kubebuilder:validation:Required
	K8S *ProxyK8sConfig `json:"k8s"`
}

// **ProxyK8sConfig** is a subset of [KubernetesResourcesSpec](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec).
type ProxyK8sConfig struct {
	// Defines Kubernetes resources' configuration. See [Resource Management for Pods and Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).
	Resources *Resources `json:"resources,omitempty"`
}

// Configures the Istio CNI DaemonSet component.
type CniComponent struct {
	// Configures the Istio CNI DaemonSet component. It is a subset of [KubernetesResourcesSpec](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec).
	// +kubebuilder:validation:Required
	K8S *CniK8sConfig `json:"k8s"`
}

// Configures the Istio CNI DaemonSet component. It is a subset of [KubernetesResourcesSpec](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec).
type CniK8sConfig struct {
	// Defines the Pod scheduling affinity constraints. See [Affinity and anti-affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity).
	// +kubebuilder:validation:Optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Defines Kubernetes resources' configuration. See [Resource Management for Pods and Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).
	// +kubebuilder:validation:Optional
	Resources *Resources `json:"resources,omitempty"`
}

// Configures the [HorizontalPodAutoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/).
type HPASpec struct {
	// Defines the minimum number of replicas for the HorizontalPodAutoscaler.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2147483647
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// Defines the maximum number of replicas for the HorizontalPodAutoscaler.
	// +kubebuilder:validation:Minimum=0
	// +kubebuilder:validation:Maximum=2147483647
	MinReplicas *int32 `json:"minReplicas,omitempty"`
}

// Defines the configuration for the generic Istio components, that is, Istio Ingress gateway and istiod.
type IstioComponent struct {
	// Defines the Kubernetes resources' configuration for Istio components. It's a subset of [KubernetesResourcesSpec](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec).
	// +kubebuilder:validation:Required
	K8s *KubernetesResourcesConfig `json:"k8s"`
}

// Defines the rolling updates strategy. See [Rolling Update Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment).
type Strategy struct {
	// Defines the configuration for rolling updates. See [Rolling Update Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment).
	// +kubebuilder:validation:Required
	RollingUpdate *RollingUpdate `json:"rollingUpdate"`
}

// Defines the configuration for rolling updates. See [Rolling Update Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#rolling-update-deployment).
type RollingUpdate struct {
	// Specifies the maximum number of Pods that can be unavailable during the update process. See [Max Surge](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#max-surge).
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Pattern=`^[0-9]+%?$`
	// +kubebuilder:validation:XValidation:rule="(type(self) == int ? self >= 0 && self <= 2147483647: self.size() >= 0)",message="must not be negative, more than 2147483647 or an empty string"
	MaxSurge *intstr.IntOrString `json:"maxSurge"       protobuf:"bytes,2,opt,name=maxSurge"`
	// Specifies the maximum number of Pods that can be created over the desired number of Pods. See [Max Unavailable](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#max-unavailable)
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Pattern="^((100|[0-9]{1,2})%|[0-9]+)$"
	// +kubebuilder:validation:XValidation:rule="(type(self) == int ? self >= 0 && self <= 2147483647: self.size() >= 0)",message="must not be negative, more than 2147483647 or an empty string"
	// +kubebuilder:validation:Optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable" protobuf:"bytes,1,opt,name=maxUnavailable"`
}

// Defines Kubernetes resources' configuration. See [Resource Management for Pods and Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).
type Resources struct {
	// The maximum amount of resources a container is allowed to use.
	Limits *ResourceClaims `json:"limits,omitempty"`
	// The minimum amount of resources (such as CPU and memory) a container needs to run.
	Requests *ResourceClaims `json:"requests,omitempty"`
}

// Defines CPU and memory resource requirements for Kubernetes containers and Pods. See [Resource Management for Pods and Containers](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/).
type ResourceClaims struct {
	// Specifies CPU resource allocation (requests or limits)
	// +kubebuilder:validation:Pattern=`^([0-9]+m?|[0-9]\.[0-9]{1,3})$`
	CPU *string `json:"cpu,omitempty"`
	// Specifies memory resource allocation (requests or limits).
	// +kubebuilder:validation:Pattern=`^[0-9]+(((\.[0-9]+)?(E|P|T|G|M|k|Ei|Pi|Ti|Gi|Mi|Ki|m)?)|(e[0-9]+))$`
	Memory *string `json:"memory,omitempty"`
}

// Configures the Istio Egress Gateway component.
type EgressGateway struct {
	// Defines the Kubernetes resources' configuration for Istio Egress Gateway. It's a subset of [KubernetesResourcesSpec](https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/#KubernetesResourcesSpec).
	// +kubebuilder:validation:Optional
	K8s *KubernetesResourcesConfig `json:"k8s"`
	// Enables or disables Istio Egress Gateway.
	// +kubebuilder:validation:Optional
	Enabled *bool `json:"enabled,omitempty"`
}
