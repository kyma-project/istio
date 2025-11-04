package istiocrmetrics

import (
	"github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

type IstioCRMetrics struct {
	extAuthMetrics *extAuthMetrics
}

type extAuthMetrics struct {
	ProvidersTotal                  prometheus.Gauge
	TimeoutConfiguredNumberTotal    prometheus.Gauge
	PathPrefixConfiguredNumberTotal prometheus.Gauge
}

//type gatewayMetrics struct {
//	gatewayExternalTrafficPolicyEnabled *prometheus.Gauge
//}

func RegisterMetrics() *IstioCRMetrics {
	crMetrics := &IstioCRMetrics{
		extAuthMetrics: &extAuthMetrics{
			ProvidersTotal: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_ext_auth_providers_total",
				Help: "Total number of external authorization providers defined in the Istio CR.",
			}),
			TimeoutConfiguredNumberTotal: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_ext_auth_timeout_configured_number_total",
				Help: "Total number of external authorization providers with timeout configured in the Istio CR.",
			}),
			PathPrefixConfiguredNumberTotal: prometheus.NewGauge(prometheus.GaugeOpts{
				Name: "istio_ext_auth_path_prefix_configured_number_total",
				Help: "Total number of external authorization providers with path prefix configured in the Istio CR.",
			}),
		},
	}

	ctrlmetrics.Registry.MustRegister(
		crMetrics.extAuthMetrics.ProvidersTotal,
		crMetrics.extAuthMetrics.TimeoutConfiguredNumberTotal,
		crMetrics.extAuthMetrics.PathPrefixConfiguredNumberTotal,
	)

	return crMetrics
}

func (m *IstioCRMetrics) EmitIstioCRMetrics(cr *v1alpha2.Istio) {
	providersTotal := len(cr.Spec.Config.Authorizers)
	m.extAuthMetrics.ProvidersTotal.Set(float64(providersTotal))

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

	m.extAuthMetrics.TimeoutConfiguredNumberTotal.Set(float64(timeoutCount))
	m.extAuthMetrics.PathPrefixConfiguredNumberTotal.Set(float64(pathPrefixCount))
}
