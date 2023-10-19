package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrl "sigs.k8s.io/controller-runtime"
	k8sMetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var installationCounter = prometheus.NewCounter(prometheus.CounterOpts{
	Name:        "installations_processed",
	Help:        "Number of installations that istio controller processed",
	ConstLabels: prometheus.Labels{"module": "istio"},
})

func Initialise() {
	k8sMetrics.Registry.MustRegister(installationCounter)
}

func IncrementInstallations() {
	ctrl.Log.Info("Incrementing installations")
	installationCounter.Inc()
}
