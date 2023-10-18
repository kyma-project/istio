package metrics

import "github.com/prometheus/client_golang/prometheus"

var installationCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name: "installations_processed",
	Help: "Number of installations that istio controller processed",
})

func init() {
	prometheus.MustRegister(installationCounter)
}

func IncrementInstallations() {
	installationCounter.Inc()
}
