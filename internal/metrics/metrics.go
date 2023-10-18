package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	namespace = "kyma"
	subsystem = "kyma_istio_operator"
)

func CreateInstallationsCounter() prometheus.Counter {
	requestCounterOpts := prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "installations_processed",
		Help:      "Number of installations that istio controller processed",
	}
	installationCounter := prometheus.NewCounter(requestCounterOpts)
	prometheus.MustRegister(installationCounter)

	return installationCounter
}
