package v1alpha2

// Configures Istio telemetry.
type Telemetry struct {
	// Configures Istio telemetry metrics.
	// +kubebuilder:validation:Optional
	Metrics Metrics `json:"metrics,omitempty"`
}

// Configures Istio telemetry metrics.
type Metrics struct {
	// Defines whether the **prometheusMerge** feature is enabled. If it is, appropriate prometheus.io annotations are added to all data plane Pods to set up scraping.
	// If these annotations already exist, they are overwritten. With this option, the Envoy sidecar merges Istio’s metrics with the application metrics.
	// The merged metrics are scraped from `:15020/stats/prometheus`.
	// +kubebuilder:validation:Optional
	PrometheusMerge bool `json:"prometheusMerge,omitempty"`
}