package v1alpha1

// Configurion.
type Config struct {
	// Defines the number of trusted proxies deployed in front of the Istio gateway proxy.
	// +kubebuilder:validation:Optional
	NumTrustedProxies int `json:"numTrustedProxies,omitempty"`
}
