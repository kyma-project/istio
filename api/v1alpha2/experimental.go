package v1alpha2

// Defines experimental features.
type Experimental struct {
	// Defines experimental features for Istio Pilot.
	// +kubebuilder:validation:Optional
	PilotFeatures `json:"pilot"`

	// Enables ambient mode support.
	// +kubebuilder:validation:Optional
	EnableAmbient *bool `json:"enableAmbient,omitempty"`
// TODO(gateway-api-parked): This feature was parked in April 2026.
// Before resuming: migrate EnableGatewayAPI *bool → GatewayAPIConfig struct
// for extensibility. See docs/contributor/adr/0017-support-for-gateway-api.md
	
	// Enables installation of Gateway API CRDs.
	// When set to true, the Gateway API CRDs are installed as part of the Istio installation.
	// When set to false or unset, the Gateway API CRDs are not installed.
	// +kubebuilder:validation:Optional
	EnableGatewayAPI *bool `json:"enableGatewayAPI,omitempty"`
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
