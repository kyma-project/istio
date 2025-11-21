package v1alpha2

type Experimental struct {
	PilotFeatures `json:"pilot"`

	// Enables the Dual Stack support
	// +kubebuilder:validation:Optional
	EnableDualStack *bool `json:"enableDualStack,omitempty"`
	EnableAmbient   *bool `json:"enableAmbient,omitempty"`
}

type PilotFeatures struct {
	EnableAlphaGatewayAPI                bool `json:"enableAlphaGatewayAPI"`
	EnableMultiNetworkDiscoverGatewayAPI bool `json:"enableMultiNetworkDiscoverGatewayAPI"`
}
