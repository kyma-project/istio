package controllers

import (
	"time"

	"github.com/kyma-project/module-manager/operator/pkg/declarative"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IstioReconciler reconciles a Istio object
type IstioReconciler struct {
	declarative.ManifestReconciler // declarative reconciler override
	*rest.Config                   // required to pass rest config to the declarative library
	client.Client
	Scheme *runtime.Scheme
}

// ManifestResolver represents the chart information for the passed Istio resource.
type ManifestResolver struct {
	chartPath string
}

type RateLimiter struct {
	Burst           int
	Frequency       int
	BaseDelay       time.Duration
	FailureMaxDelay time.Duration
}
