package controllers

import (
	"context"
	"time"

	"github.com/kyma-project/istio/operator/internal/resources"
	"github.com/kyma-project/istio/operator/internal/restarter"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/internal/describederrors"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istioresources"
	"github.com/kyma-project/istio/operator/internal/status"
)

type Reconciliation interface {
	Reconcile(ctx context.Context) describederrors.DescribedError
}

// IstioReconciler reconciles a Istio object.
type IstioReconciler struct {
	*rest.Config // required to pass rest config to the declarative library
	client.Client
	Scheme                 *runtime.Scheme
	istioInstallation      istio.InstallationReconciliation
	istioResources         istioresources.ResourcesReconciliation
	userResources          resources.UserResourcesFinder
	restarters             []restarter.Restarter
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
