package metrics

import "github.com/prometheus/client_golang/prometheus"

func CreateInstallationsCounter() prometheus.Counter {
	requestCounterOpts := prometheus.CounterOpts{
		Name: "installations_processed",
		Help: "Number of installations that istio controller processed",
	}
	installationCounter := prometheus.NewCounter(requestCounterOpts)
	prometheus.MustRegister(installationCounter)

	return installationCounter
}
