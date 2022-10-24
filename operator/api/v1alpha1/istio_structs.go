package v1alpha1

// Type definitions are based on https://istio.io/latest/docs/reference/config/istio.operator.v1alpha1/

type GatewayTopology struct {
	// +kubebuilder:validation:Optional
	NumTrustedProxies int `json:"numTrustedProxies,omitempty"`
}

type MeshConfig struct {
	// +kubebuilder:validation:Optional
	GatewayTopology GatewayTopology `json:"gatewayTopology,omitempty"`
}

type ResourceMetricSource struct {
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// +kubebuilder:validation:Optional
	TargetAverageUtilization int `json:"targetAverageUtilization,omitempty"`
}

type MetricSpec struct {
	// +kubebuilder:validation:Optional
	Type string `json:"type,omitempty"`

	// +kubebuilder:validation:Optional
	Resource ResourceMetricSource `json:"resource,omitempty"`
}

type HorizontalPodAutoscalerSpec struct {
	// +kubebuilder:validation:Optional
	MaxReplicas int `json:"maxReplicas,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=1
	MinReplicas int `json:"minReplicas,omitempty"`

	// +kubebuilder:validation:Optional
	Metrics []MetricSpec `json:"metrics,omitempty"`
}

type RollingUpdateDeployment struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[1-9][\d]?%|100%|\d+$
	MaxSurge string `json:"maxSurge,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[1-9][\d]?%|100%|\d+$
	MaxUnavailable string `json:"maxUnavailable,omitempty"`
}

type DeploymentStrategy struct {
	// +kubebuilder:validation:Optional
	RollingUpdate RollingUpdateDeployment `json:"rollingUpdate,omitempty"`
}

type ResourceSpec struct {
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[1-9](\d?)+(m|g)$
	CPU string `json:"cpu,omitempty"`

	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Pattern=^[1-9](\d?)+(Mi|Gi)$
	Memory string `json:"memory,omitempty"`
}

type Resources struct {
	// +kubebuilder:validation:Optional
	Limits ResourceSpec `json:"limits,omitempty"`

	// +kubebuilder:validation:Optional
	Requests ResourceSpec `json:"requests,omitempty"`
}

type Deployment struct {
	// +kubebuilder:validation:Optional
	HpaSpec HorizontalPodAutoscalerSpec `json:"hpa,omitempty"`

	// +kubebuilder:validation:Optional
	Strategy DeploymentStrategy `json:"strategy,omitempty"`

	// +kubebuilder:validation:Optional
	Resources Resources `json:"resources,omitempty"`
}

type Istiod struct {
	// +kubebuilder:validation:Optional
	Deployment Deployment `json:"deployment,omitempty"`
}

type Controlplane struct {
	// +kubebuilder:validation:Optional
	MeshConfig MeshConfig `json:"meshConfig,omitempty"`

	// +kubebuilder:validation:Optional
	Istiod Istiod `json:"istiod,omitempty"`
}

type IngressGateway struct {
	// +kubebuilder:validation:Optional
	Deployment Deployment `json:"deployment,omitempty"`
}

type Dataplane struct {
	// +kubebuilder:validation:Optional
	IngressGateway IngressGateway `json:"ingressGateway,omitempty"`
}
