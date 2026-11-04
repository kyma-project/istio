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

// nolint:gocyclo
func main() {
	var metricsAddr string
	var metricsCertPath, metricsCertName, metricsCertKey string
	var webhookCertPath, webhookCertName, webhookCertKey string
	var webhookPort int
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var istioCRNamespace, istioCRName string
	var proxyCPURequest, proxyMemoryRequest, proxyCPULimit, proxyMemoryLimit string
	var devMode bool
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.StringVar(&webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	flag.StringVar(&webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	flag.StringVar(&webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
	flag.IntVar(&webhookPort, "webhook-port", 9443, "Port the webhook server listens on. "+
		"Defaults to 9443. Set -1 to disable the webhook server.")
	flag.StringVar(&metricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	flag.StringVar(&metricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	flag.StringVar(&metricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&istioCRNamespace, "istio-cr-namespace", "kyma-system", "Namespace of the Istio CR to read evaluators from.")
	flag.StringVar(&istioCRName, "istio-cr-name", "default", "Name of the Istio CR to read evaluators from.")
	flag.StringVar(&proxyCPURequest, "proxy-cpu-request", "", "Expected CPU request for the Istio proxy sidecar container (required).")
	flag.StringVar(&proxyMemoryRequest, "proxy-memory-request", "", "Expected memory request for the Istio proxy sidecar container (required).")
	flag.StringVar(&proxyCPULimit, "proxy-cpu-limit", "", "Expected CPU limit for the Istio proxy sidecar container (required).")
	flag.StringVar(&proxyMemoryLimit, "proxy-memory-limit", "", "Expected memory limit for the Istio proxy sidecar container (required).")
	flag.BoolVar(&devMode, "dev-mode", false,
		"If set, use built-in development sidecar resource values instead of the required --proxy-* flags.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	var sidecarResources corev1.ResourceRequirements
	if devMode {
		setupLog.Info("Dev mode: using built-in sidecar resource values, ignoring --proxy-* flags")
		sidecarResources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("10m"),
				corev1.ResourceMemory: resource.MustParse("32Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("250m"),
				corev1.ResourceMemory: resource.MustParse("254Mi"),
			},
		}
	} else {
		if proxyCPURequest == "" {
			setupLog.Error(nil, "Flag --proxy-cpu-request is required")
			os.Exit(1)
		}
		cpuReq, err := resource.ParseQuantity(proxyCPURequest)
		if err != nil {
			setupLog.Error(err, "Failed to parse --proxy-cpu-request")
			os.Exit(1)
		}

		if proxyMemoryRequest == "" {
			setupLog.Error(nil, "Flag --proxy-memory-request is required")
			os.Exit(1)
		}
		memReq, err := resource.ParseQuantity(proxyMemoryRequest)
		if err != nil {
			setupLog.Error(err, "Failed to parse --proxy-memory-request")
			os.Exit(1)
		}

		if proxyCPULimit == "" {
			setupLog.Error(nil, "Flag --proxy-cpu-limit is required")
			os.Exit(1)
		}
		cpuLim, err := resource.ParseQuantity(proxyCPULimit)
		if err != nil {
			setupLog.Error(err, "Failed to parse --proxy-cpu-limit")
			os.Exit(1)
		}

		if proxyMemoryLimit == "" {
			setupLog.Error(nil, "Flag --proxy-memory-limit is required")
			os.Exit(1)
		}
		memLim, err := resource.ParseQuantity(proxyMemoryLimit)
		if err != nil {
			setupLog.Error(err, "Failed to parse --proxy-memory-limit")
			os.Exit(1)
		}

		sidecarResources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: cpuReq, corev1.ResourceMemory: memReq},
			Limits:   corev1.ResourceList{corev1.ResourceCPU: cpuLim, corev1.ResourceMemory: memLim},
		}
	}

	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("Disabling HTTP/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	// Initial webhook TLS options
	webhookTLSOpts := tlsOpts
	webhookServerOptions := webhook.Options{
		TLSOpts: webhookTLSOpts,
		Port:    webhookPort,
	}

	if len(webhookCertPath) > 0 {
		setupLog.Info("Initializing webhook certificate watcher using provided certificates",
			"webhook-cert-path", webhookCertPath, "webhook-cert-name", webhookCertName, "webhook-cert-key", webhookCertKey)

		webhookServerOptions.CertDir = webhookCertPath
		webhookServerOptions.CertName = webhookCertName
		webhookServerOptions.KeyName = webhookCertKey
	}

	webhookServer := webhook.NewServer(webhookServerOptions)

	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	if len(metricsCertPath) > 0 {
		setupLog.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", metricsCertPath, "metrics-cert-name", metricsCertName, "metrics-cert-key", metricsCertKey)

		metricsServerOptions.CertDir = metricsCertPath
		metricsServerOptions.CertName = metricsCertName
		metricsServerOptions.KeyName = metricsCertKey
	}

	stripTransform := transform.StripForRestart()
	stripKeepOwnerRefs := transform.StripForRestartKeepOwnerRefs()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "02e73aaf.kyma-project.io",
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&appsv1.Deployment{}:  {Transform: stripTransform},
				&appsv1.ReplicaSet{}:  {Transform: stripKeepOwnerRefs},
				&appsv1.StatefulSet{}: {Transform: stripTransform},
				&appsv1.DaemonSet{}:   {Transform: stripTransform},
				&corev1.Pod{}:         {Transform: stripKeepOwnerRefs},
			},
		},
	})
	if err != nil {
		setupLog.Error(err, "Failed to start manager")
		os.Exit(1)
	}

	evalSource := evaluation.NewProvider(
		mgr.GetClient(),
		types.NamespacedName{Namespace: istioCRNamespace, Name: istioCRName},
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
