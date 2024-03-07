package controllers

import (
	"context"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"time"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	"github.com/kyma-project/istio/operator/internal/status"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciliation interface {
	Reconcile(ctx context.Context) described_errors.DescribedError
}

type Restarter interface {
	Restart(ctx context.Context, istioCR *operatorv1alpha2.Istio) described_errors.DescribedError
}

// IstioReconciler reconciles a Istio object
type IstioReconciler struct {
	*rest.Config // required to pass rest config to the declarative library
	client.Client
	Scheme                 *runtime.Scheme
	istioInstallation      istio.InstallationReconciliation
	istioResources         istio_resources.ResourcesReconciliation
	restarters             []Restarter
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
