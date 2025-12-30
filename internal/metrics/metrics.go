package istiocrmetrics

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kyma-project/istio/operator/api/v1alpha2"
)

// IstioCRMetrics holds all the metrics related to the Istio CR.
// It includes metrics for external authorization, configuration, and components.
// The 'registered' field indicates whether the metrics have been registered with Prometheus.
type IstioCRMetrics struct {
	extAuthMetrics   *extAuthMetrics
	configMetrics    *configMetrics
	componentMetrics *componentMetrics
}

type configMetrics struct {
	numTrustedProxiesConfigured        prometheus.Gauge
	prometheusMergeEnabled             prometheus.Gauge
	compatibilityModeEnabled           prometheus.Gauge
	forwardClientCertDetailsConfigured prometheus.Gauge
	trustDomainConfigured              prometheus.Gauge
}

type componentMetrics struct {
	egressGatewayEnabled prometheus.Gauge
}

type extAuthMetrics struct {
	providersTotal                  prometheus.Gauge
	timeoutConfiguredNumberTotal    prometheus.Gauge
	pathPrefixConfiguredNumberTotal prometheus.Gauge
}

func NewMetrics() *IstioCRMetrics {
	crMetrics := &IstioCRMetrics{
		extAuthMetrics: &extAuthMetrics{
			providersTotal: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_ext_auth_providers_total",
				Help: "Total number of external authorization providers defined in the Istio CR.",
			}),
			timeoutConfiguredNumberTotal: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_ext_auth_timeout_configured_number_total",
				Help: "Total number of external authorization providers with timeout configured in the Istio CR.",
			}),
			pathPrefixConfiguredNumberTotal: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_ext_auth_path_prefix_configured_number_total",
				Help: "Total number of external authorization providers with path prefix configured in the Istio CR.",
			}),
		},
		configMetrics: &configMetrics{
			numTrustedProxiesConfigured: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_num_trusted_proxies_configured",
				Help: "Indicates whether numTrustedProxies is configured in the Istio CR (1 for configured, 0 for not configured).",
			}),
			prometheusMergeEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_prometheus_merge_enabled",
				Help: "Indicates whether Prometheus merge is enabled in the Istio CR (1 for enabled, 0 for disabled).",
			}),
			compatibilityModeEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_compatibility_mode_enabled",
				Help: "Indicates whether compatibility mode is enabled in the Istio CR (1 for enabled, 0 for disabled).",
			}),
			forwardClientCertDetailsConfigured: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_forward_client_cert_details_configured",
				Help: "Indicates whether forwardClientCertDetails is configured in the Istio CR (1 for configured, 0 for not configured).",
			}),
			trustDomainConfigured: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_trust_domain_configured",
				Help: "Indicates whether a custom trust domain is configured in the Istio CR (1 for configured, 0 for not configured).",
			}),
		},
		componentMetrics: &componentMetrics{
			egressGatewayEnabled: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_egress_gateway_used",
				Help: "Indicates whether the egress gateway is used in the Istio CR (1 for used, 0 for not used).",
			}),
		},
	}

	ctrlmetrics.Registry.MustRegister(
		crMetrics.extAuthMetrics.providersTotal,
		crMetrics.extAuthMetrics.timeoutConfiguredNumberTotal,
		crMetrics.extAuthMetrics.pathPrefixConfiguredNumberTotal,
		crMetrics.configMetrics.numTrustedProxiesConfigured,
		crMetrics.configMetrics.prometheusMergeEnabled,
		crMetrics.configMetrics.compatibilityModeEnabled,
		crMetrics.configMetrics.forwardClientCertDetailsConfigured,
		crMetrics.componentMetrics.egressGatewayEnabled,
	)

	return crMetrics
}

func (m *IstioCRMetrics) UpdateIstioCRMetrics(cr *v1alpha2.Istio) {
	providersTotal := len(cr.Spec.Config.Authorizers)
	m.extAuthMetrics.providersTotal.Set(float64(providersTotal))

	timeoutCount := 0
	pathPrefixCount := 0
	for _, authorizer := range cr.Spec.Config.Authorizers {
		if authorizer.PathPrefix != nil {
			pathPrefixCount++
		}
		if authorizer.Timeout != nil {
			timeoutCount++
		}
	}

	m.extAuthMetrics.timeoutConfiguredNumberTotal.Set(float64(timeoutCount))
	m.extAuthMetrics.pathPrefixConfiguredNumberTotal.Set(float64(pathPrefixCount))

	if cr.Spec.Config.NumTrustedProxies != nil && *cr.Spec.Config.NumTrustedProxies != 0 {
		m.configMetrics.numTrustedProxiesConfigured.Set(1)
	} else {
		m.configMetrics.numTrustedProxiesConfigured.Set(0)
	}

	if cr.Spec.Config.Telemetry.Metrics.PrometheusMerge {
		m.configMetrics.prometheusMergeEnabled.Set(1)
	} else {
		m.configMetrics.prometheusMergeEnabled.Set(0)
	}

	if cr.Spec.Config.ForwardClientCertDetails != nil {
		m.configMetrics.forwardClientCertDetailsConfigured.Set(1)
	} else {
		m.configMetrics.forwardClientCertDetailsConfigured.Set(0)
	}

	if cr.Spec.Config.TrustDomain != nil && *cr.Spec.Config.TrustDomain != "" && *cr.Spec.Config.TrustDomain != "cluster.local" {
		m.configMetrics.trustDomainConfigured.Set(1)
	} else {
		m.configMetrics.trustDomainConfigured.Set(0)
	}

	if cr.Spec.CompatibilityMode {
		m.configMetrics.compatibilityModeEnabled.Set(1)
	} else {
		m.configMetrics.compatibilityModeEnabled.Set(0)
	}

	if cr.Spec.Components != nil && cr.Spec.Components.EgressGateway != nil &&
		cr.Spec.Components.EgressGateway.Enabled != nil &&
		*cr.Spec.Components.EgressGateway.Enabled {
		m.componentMetrics.egressGatewayEnabled.Set(1)
	} else {
		m.componentMetrics.egressGatewayEnabled.Set(0)
	}

}
