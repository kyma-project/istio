package v1alpha2

// Defines experimental features.
type Experimental struct {
	// Defines experimental features for Istio Pilot.
	// +kubebuilder:validation:Optional
	PilotFeatures `json:"pilot"`

	// Enables dual-stack support.
	// +kubebuilder:validation:Optional
	EnableDualStack *bool `json:"enableDualStack,omitempty"`
	// Enables ambient mode support.
	// +kubebuilder:validation:Optional
	EnableAmbient *bool `json:"enableAmbient,omitempty"`
}

// Defines experimental features for Istio Pilot.
type PilotFeatures struct {
	// Defines alpha Gateway API support.
	// +kubebuilder:validation:Optional
	EnableAlphaGatewayAPI bool `json:"enableAlphaGatewayAPI"`
	// Enables multi-network discovery for Gateway API.
	// +kubebuilder:validation:Optional
	EnableMultiNetworkDiscoverGatewayAPI bool `json:"enableMultiNetworkDiscoverGatewayAPI"`
}
