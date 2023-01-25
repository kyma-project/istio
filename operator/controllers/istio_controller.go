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

package controllers

import (
	"context"
	"time"

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"

	"github.com/kyma-project/module-manager/operator/pkg/declarative"
	"github.com/kyma-project/module-manager/operator/pkg/types"
)

var (
	defaultIstioOperatorPath = "manifests/default-istio-operator-k3d.yaml"
	workingDir               = "/tmp"
)

const (
	IstioVersion   string = "1.16.1"
	IstioImageBase string = "distroless"
)

func NewReconciler(mgr manager.Manager) *IstioReconciler {
	return &IstioReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		istioInstallation: istio.Installation{Client: istio.NewIstioClient(defaultIstioOperatorPath, workingDir), IstioVersion: IstioVersion, IstioImageBase: IstioImageBase},
	}
}

func (r *IstioReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Was called to reconcile Kyma Istio Service Mesh")

	istioCR := operatorv1alpha1.Istio{}
	if err := r.Client.Get(ctx, req.NamespacedName, &istioCR); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Error during fetching Istio CR")
		return ctrl.Result{}, err
	}

	if err := r.istioInstallation.Reconcile(ctx, &istioCR, r.Client); err != nil {
		logger.Error(err, "Error occurred during reconciliation of Istio Operator")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
}

// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;create;update;patch
func (r *IstioReconciler) SetupWithManager(mgr ctrl.Manager, chartPath string, configFlags, setFlags types.Flags, rateLimiter RateLimiter) error {
	ConfigFlags = configFlags
	SetFlags = setFlags
	r.Config = mgr.GetConfig()
	if err := r.initReconciler(mgr, chartPath); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Istio{}).
		WithOptions(controller.Options{
			RateLimiter: TemplateRateLimiter(
				rateLimiter.BaseDelay,
				rateLimiter.FailureMaxDelay,
				rateLimiter.Frequency,
				rateLimiter.Burst,
			),
		}).
		Complete(r)
}

func (r *IstioReconciler) initReconciler(mgr ctrl.Manager, chartPath string) error {
	manifestResolver := &ManifestResolver{chartPath: chartPath}
	return r.Inject(mgr, &operatorv1alpha1.Istio{},
		declarative.WithManifestResolver(manifestResolver),
		declarative.WithCustomResourceLabels(map[string]string{istioAnnotationKey: istioAnnotationValue}),
		declarative.WithPostRenderTransform(transform),
		declarative.WithResourcesReady(true),
		declarative.WithFinalizer(istioFinalizer),
	)
}

// TemplateRateLimiter implements a rate limiter for a client-go.workqueue.  It has
// both an overall (token bucket) and per-item (exponential) rate limiting.
func TemplateRateLimiter(failureBaseDelay time.Duration, failureMaxDelay time.Duration,
	frequency int, burst int,
) ratelimiter.RateLimiter {
	return workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(failureBaseDelay, failureMaxDelay),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(frequency), burst)})
}
