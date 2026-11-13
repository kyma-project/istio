/*
Copyright 2026.

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
	"crypto/tls"
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/restarter/internal/controller"
	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	"github.com/kyma-project/istio/operator/restarter/internal/transform"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha2.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

type config struct {
	metricsAddr          string
	metricsCertPath      string
	metricsCertName      string
	metricsCertKey       string
	webhookCertPath      string
	webhookCertName      string
	webhookCertKey       string
	webhookPort          int
	enableLeaderElection bool
	probeAddr            string
	secureMetrics        bool
	enableHTTP2          bool
	istioCRNamespace     string
	istioCRName          string
	proxyCPURequest      string
	proxyMemoryRequest   string
	proxyCPULimit        string
	proxyMemoryLimit     string
}

func parseFlags() (config, zap.Options) {
	var cfg config
	flag.StringVar(&cfg.metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&cfg.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&cfg.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&cfg.secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&cfg.webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&cfg.webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&cfg.webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.IntVar(&cfg.webhookPort, "webhook-port", 9443, "Port the webhook server listens on. "+
		"Defaults to 9443. Set -1 to disable the webhook server.")
	flag.StringVar(&cfg.metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&cfg.metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&cfg.metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.BoolVar(&cfg.enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&cfg.istioCRNamespace, "istio-cr-namespace", "kyma-system", "Namespace of the Istio CR to read evaluators from.")
	flag.StringVar(&cfg.istioCRName, "istio-cr-name", "default", "Name of the Istio CR to read evaluators from.")
	flag.StringVar(&cfg.proxyCPURequest, "proxy-cpu-request", "", "Expected CPU request for the Istio proxy sidecar container (required).")
	flag.StringVar(&cfg.proxyMemoryRequest, "proxy-memory-request", "", "Expected memory request for the Istio proxy sidecar container (required).")
	flag.StringVar(&cfg.proxyCPULimit, "proxy-cpu-limit", "", "Expected CPU limit for the Istio proxy sidecar container (required).")
	flag.StringVar(&cfg.proxyMemoryLimit, "proxy-memory-limit", "", "Expected memory limit for the Istio proxy sidecar container (required).")
	opts := zap.Options{
		Development: false,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	return cfg, opts
}

// nolint:gocyclo
func main() {
	cfg, opts := parseFlags()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if cfg.proxyCPURequest == "" {
		setupLog.Error(nil, "Flag --proxy-cpu-request is required")
		os.Exit(1)
	}
	cpuReq, err := resource.ParseQuantity(cfg.proxyCPURequest)
	if err != nil {
		setupLog.Error(err, "Failed to parse --proxy-cpu-request")
		os.Exit(1)
	}

	if cfg.proxyMemoryRequest == "" {
		setupLog.Error(nil, "Flag --proxy-memory-request is required")
		os.Exit(1)
	}
	memReq, err := resource.ParseQuantity(cfg.proxyMemoryRequest)
	if err != nil {
		setupLog.Error(err, "Failed to parse --proxy-memory-request")
		os.Exit(1)
	}

	if cfg.proxyCPULimit == "" {
		setupLog.Error(nil, "Flag --proxy-cpu-limit is required")
		os.Exit(1)
	}
	cpuLim, err := resource.ParseQuantity(cfg.proxyCPULimit)
	if err != nil {
		setupLog.Error(err, "Failed to parse --proxy-cpu-limit")
		os.Exit(1)
	}

	if cfg.proxyMemoryLimit == "" {
		setupLog.Error(nil, "Flag --proxy-memory-limit is required")
		os.Exit(1)
	}
	memLim, err := resource.ParseQuantity(cfg.proxyMemoryLimit)
	if err != nil {
		setupLog.Error(err, "Failed to parse --proxy-memory-limit")
		os.Exit(1)
	}

	sidecarResources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{corev1.ResourceCPU: cpuReq, corev1.ResourceMemory: memReq},
		Limits:   corev1.ResourceList{corev1.ResourceCPU: cpuLim, corev1.ResourceMemory: memLim},
	}

	var tlsOpts []func(*tls.Config)
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("Disabling HTTP/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !cfg.enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	// Initial webhook TLS options
	webhookTLSOpts := tlsOpts
	webhookServerOptions := webhook.Options{
		TLSOpts: webhookTLSOpts,
		Port:    cfg.webhookPort,
	}

	if len(cfg.webhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", cfg.webhookCertPath, "webhook-cert-name", cfg.webhookCertName, "webhook-cert-key", cfg.webhookCertKey)

		webhookServerOptions.CertDir = cfg.webhookCertPath
		webhookServerOptions.CertName = cfg.webhookCertName
		webhookServerOptions.KeyName = cfg.webhookCertKey
	}

	webhookServer := webhook.NewServer(webhookServerOptions)

	metricsServerOptions := metricsserver.Options{
		BindAddress:   cfg.metricsAddr,
		SecureServing: cfg.secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if cfg.secureMetrics {
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	if len(cfg.metricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", cfg.metricsCertPath, "metrics-cert-name", cfg.metricsCertName, "metrics-cert-key", cfg.metricsCertKey)

		metricsServerOptions.CertDir = cfg.metricsCertPath
		metricsServerOptions.CertName = cfg.metricsCertName
		metricsServerOptions.KeyName = cfg.metricsCertKey
	}

	stripTransform := transform.StripForRestart()
	stripKeepOwnerRefs := transform.StripForRestartKeepOwnerRefs()
	stripPod := transform.StripForPod()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: cfg.probeAddr,
		LeaderElection:         cfg.enableLeaderElection,
		LeaderElectionID:       "02e73aaf.kyma-project.io",
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&appsv1.Deployment{}:  {Transform: stripTransform},
				&appsv1.ReplicaSet{}:  {Transform: stripKeepOwnerRefs},
				&appsv1.StatefulSet{}: {Transform: stripTransform},
				&appsv1.DaemonSet{}:   {Transform: stripTransform},
				&corev1.Pod{}:         {Transform: stripPod},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "Failed to start manager")
		os.Exit(1)
	}

	evalSource := evaluation.NewProvider(
		mgr.GetClient(),
		types.NamespacedName{Namespace: cfg.istioCRNamespace, Name: cfg.istioCRName},
		sidecarResources,
	)

	if err := (&controller.DeploymentReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Evaluators: evalSource,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create controller", "controller", "deployment")
		os.Exit(1)
	}
	if err := (&controller.ReplicaSetReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Evaluators: evalSource,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create controller", "controller", "replicaset")
		os.Exit(1)
	}
	if err := (&controller.StatefulSetReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Evaluators: evalSource,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create controller", "controller", "statefulset")
		os.Exit(1)
	}
	if err := (&controller.DaemonSetReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Evaluators: evalSource,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "Failed to create controller", "controller", "daemonset")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "Failed to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "Failed to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("Starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "Failed to run manager")
		os.Exit(1)
	}
}
