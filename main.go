/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"time"

	networkingv1alpha3 "istio.io/client-go/pkg/apis/networking/v1alpha3"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"

	networkingv1 "istio.io/client-go/pkg/apis/networking/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/controllers"
	//+kubebuilder:scaffold:imports
)

const (
	rateLimiterBurstDefault       = 200
	rateLimiterFrequencyDefault   = 30
	failureBaseDelayDefault       = 1 * time.Second
	failureMaxDelayDefault        = 1000 * time.Second
	reconciliationIntervalDefault = 10 * time.Hour

	WebhookServiceDefaultPort = 9443
)

//nolint:gochecknoglobals // it was scaffolded by controller-gen TODO: remove this global variable when possible
var (
	scheme = runtime.NewScheme()
)

type FlagVar struct {
	metricsAddr            string
	enableLeaderElection   bool
	probeAddr              string
	failureBaseDelay       time.Duration
	failureMaxDelay        time.Duration
	rateLimiterFrequency   int
	rateLimiterBurst       int
	reconciliationInterval time.Duration
}

func init() { //nolint:gochecknoinits // it was scaffolded by controller-gen TODO: remove this init function when possible
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(networkingv1alpha3.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha2.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	flagVar := defineFlagVar()
	setupLog := ctrl.Log.WithName("setup")
	opts := zap.Options{
		Development: true,
	}

	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	rateLimiter := controllers.RateLimiter{
		Burst:           flagVar.rateLimiterBurst,
		Frequency:       flagVar.rateLimiterFrequency,
		BaseDelay:       flagVar.failureBaseDelay,
		FailureMaxDelay: flagVar.failureMaxDelay,
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// We configure the Istio logging here to make it visible that global log config is updated instead of hiding it in the scope of istio package.
	err := istio.ConfigureIstioLogScopes()
	if err != nil {
		setupLog.Error(err, "Unable to configure Istio log scopes")
		os.Exit(1)
	}

	mgr, err := createManager(flagVar)
	if err != nil {
		setupLog.Error(err, "Unable to create manager")
		os.Exit(1)
	}

	if err = controllers.NewController(mgr, flagVar.reconciliationInterval).SetupWithManager(mgr, rateLimiter); err != nil {
		setupLog.Error(err, "Unable to create controller", "controller", "Istio")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err = mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up health check")
		os.Exit(1)
	}
	if err = mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "Unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err = mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Problem running manager")
		os.Exit(1)
	}
}

func createManager(flagVar *FlagVar) (manager.Manager, error) {
	webhookServer := webhook.NewServer(webhook.Options{
		Port: WebhookServiceDefaultPort,
	})

	return ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: flagVar.metricsAddr,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: flagVar.probeAddr,
		LeaderElection:         flagVar.enableLeaderElection,
		LeaderElectionID:       "76223278.kyma-project.io",
		Client: client.Options{
			Cache: &client.CacheOptions{
				// The cache is disabled for these objects to avoid huge memory usage.
				// Having the cache enabled had previously caused memory usage
				// to have a significant peak when sidecar restart was triggered.
				DisableFor: []client.Object{
					&v1.DaemonSet{},
					&v1.Deployment{},
					&v1.StatefulSet{},
					&v1.ReplicaSet{},
					&corev1.Pod{}, // this is required for the sidecar restart when listing pods with limit
				},
			},
		},
	})
}

func defineFlagVar() *FlagVar {
	flagVar := new(FlagVar)
	flag.StringVar(&flagVar.metricsAddr, "metrics-bind-address", ":8090", "The address the metric endpoint binds to.")
	flag.StringVar(&flagVar.probeAddr, "health-probe-bind-address", ":8091", "The address the probe endpoint binds to.")
	flag.BoolVar(&flagVar.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.IntVar(&flagVar.rateLimiterBurst, "rate-limiter-burst", rateLimiterBurstDefault,
		"Indicates the burst value for the bucket rate limiter.")
	flag.IntVar(&flagVar.rateLimiterFrequency, "rate-limiter-frequency", rateLimiterFrequencyDefault,
		"Indicates the bucket rate limiter frequency, signifying no. of events per second.")
	flag.DurationVar(&flagVar.failureBaseDelay, "failure-base-delay", failureBaseDelayDefault,
		"Indicates the failure base delay for rate limiter.")
	flag.DurationVar(&flagVar.failureMaxDelay, "failure-max-delay", failureMaxDelayDefault,
		"Indicates the failure max delay.")
	flag.DurationVar(&flagVar.reconciliationInterval, "reconciliation-interval", reconciliationIntervalDefault,
		"Indicates the time based reconciliation interval.")
	return flagVar
}
