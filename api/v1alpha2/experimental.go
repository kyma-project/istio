package v1alpha2

type Experimental struct {
	PilotFeatures `json:"pilot"`
}

type PilotFeatures struct {
	EnableAlphaGatewayAPI                bool `json:"enableAlphaGatewayAPI"`
	EnableMultiNetworkDiscoverGatewayAPI bool `json:"enableMultiNetworkDiscoverGatewayAPI"`
}
