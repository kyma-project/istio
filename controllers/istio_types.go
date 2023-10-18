package controllers

import (
	"context"
	"github.com/prometheus/client_golang/prometheus"
	"time"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	"github.com/kyma-project/istio/operator/internal/reconciliations/proxy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciliation interface {
	Reconcile(ctx context.Context) described_errors.DescribedError
}

// IstioReconciler reconciles a Istio object
type IstioReconciler struct {
	*rest.Config // required to pass rest config to the declarative library
	client.Client
	Scheme                 *runtime.Scheme
	istioInstallation      istio.InstallationReconciliation
	proxySidecars          proxy.SidecarsReconciliation
	istioResources         istio_resources.ResourcesReconciliation
	ingressGateway         Reconciliation
	log                    logr.Logger
	statusHandler          status
	reconciliationInterval time.Duration
	installationCounter    prometheus.Counter
}

type RateLimiter struct {
	Burst           int
	Frequency       int
	BaseDelay       time.Duration
	FailureMaxDelay time.Duration
}
