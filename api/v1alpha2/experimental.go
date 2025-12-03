package v1alpha2

// Defines experimental features.
type Experimental struct {
	// Defines experimental features for Istio Pilot.
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
	EnableAlphaGatewayAPI                bool `json:"enableAlphaGatewayAPI"`
	// Enables multi-network discovery for Gateway API.
	EnableMultiNetworkDiscoverGatewayAPI bool `json:"enableMultiNetworkDiscoverGatewayAPI"`
}
