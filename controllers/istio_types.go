package controllers

import (
	"github.com/kyma-project/istio/operator/internal/status"
	"time"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/internal/reconciliations/proxy"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IstioReconciler reconciles a Istio object
type IstioReconciler struct {
	*rest.Config // required to pass rest config to the declarative library
	client.Client
	Scheme                 *runtime.Scheme
	istioInstallation      istio.Installation
	proxySidecars          proxy.Sidecars
	log                    logr.Logger
	statusHandler          status.Status
	reconciliationInterval time.Duration
}

type RateLimiter struct {
	Burst           int
	Frequency       int
	BaseDelay       time.Duration
	FailureMaxDelay time.Duration
}
