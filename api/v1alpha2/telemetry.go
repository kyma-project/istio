package v1alpha2

type Telemetry struct {
	// Istio telemetry configuration related to metrics
	// +kubebuilder:validation:Optional
	Metrics Metrics `json:"metrics"`
}

type Metrics struct {
	// Defines whether the prometheusMerge feature is enabled. If yes, appropriate prometheus.io annotations will be added to all data plane pods to set up scraping.
	// If these annotations already exist, they will be overwritten. With this option, the Envoy sidecar will merge Istio’s metrics with the application metrics.
	// The merged metrics will be scraped from :15020/stats/prometheus.
	// +kubebuilder:validation:Optional
	PrometheusMerge bool `json:"prometheusMerge"`
}
