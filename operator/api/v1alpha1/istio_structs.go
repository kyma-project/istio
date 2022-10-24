package v1alpha1

type GatewayTopology struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=integer
	NumTrustedProxies int `json:"numTrustedProxies"`
}

type MeshConfig struct {
	// +kubebuilder:validation:Optional
	GatewayTopology GatewayTopology `json:"gatewayTopology,omitempty"`
}

type Hpa struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=integer
	MaxReplicas int `json:"maxReplicas"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Type=integer
	// +kubebuilder:validation:Minimum=1
	MinReplicas int `json:"minReplicas"`
}

type RollingUpdate struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[1-9][\d]?%|100%|\d+$
	MaxSurge string `json:"maxSurge"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[1-9][\d]?%|100%|\d+$
	MaxUnavailable string `json:"maxUnavailable"`
}

type Strategy struct {
	// +kubebuilder:validation:Required
	RollingUpdate RollingUpdate `json:"rollingUpdate"`
}

type CPUMemory struct {
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[1-9](\d?)+(m|g)$
	CPU string `json:"cpu"`
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^[1-9](\d?)+(Mi|Gi)$
	Memory string `json:"memory"`
}

type Resources struct {
	// +kubebuilder:validation:Required
	Limits CPUMemory `json:"limits"`
	// +kubebuilder:validation:Required
	Requests CPUMemory `json:"requests"`
}

type Deployment struct {
	// +kubebuilder:validation:Optional
	Hpa Hpa `json:"hpa,omitempty"`
	// +kubebuilder:validation:Optional
	Strategy Strategy `json:"strategy,omitempty"`
	// +kubebuilder:validation:Optional
	Resources Resources `json:"resources,omitempty"`
}

type Istiod struct {
	// +kubebuilder:validation:Required
	Deployment Deployment `json:"deployment"`
}

type Controlplane struct {
	// +kubebuilder:validation:Optional
	MeshConfig MeshConfig `json:"meshConfig,omitempty"`
	// +kubebuilder:validation:Optional
	Istiod Istiod `json:"istiod,omitempty"`
}

type Dataplane struct {
}
