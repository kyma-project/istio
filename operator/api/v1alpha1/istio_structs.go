package v1alpha1

type Config struct {
	// Configurion.
	// +kubebuilder:validation:Optional
	NumTrustedProxies int `json:"numTrustedProxies,omitempty"`
}
