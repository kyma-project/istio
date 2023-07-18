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
	"fmt"
	"time"

	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/status"
	"github.com/pkg/errors"

	"github.com/kyma-project/istio/operator/internal/manifest"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/internal/reconciliations/proxy"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"
)

const (
	namespace                    = "kyma-system"
	IstioVersion                 = "1.18.1"
	IstioImageBase               = "distroless"
	IstioResourceListDefaultPath = "manifests/controlled_resources_list.yaml"
	ErrorRetryTime               = time.Minute * 1
)

var IstioTag = fmt.Sprintf("%s-%s", IstioVersion, IstioImageBase)

func NewReconciler(mgr manager.Manager, reconciliationInterval time.Duration) *IstioReconciler {
	merger := manifest.NewDefaultIstioMerger()

	return &IstioReconciler{
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
		istioInstallation:      istio.Installation{Client: mgr.GetClient(), IstioClient: istio.NewIstioClient(), IstioVersion: IstioVersion, IstioImageBase: IstioImageBase, Merger: &merger, StatusHandler: status.NewDefaultStatusHandler()},
		proxySidecars:          proxy.Sidecars{IstioVersion: IstioVersion, IstioImageBase: IstioImageBase, Log: mgr.GetLogger(), Client: mgr.GetClient(), Merger: &merger, StatusHandler: status.NewDefaultStatusHandler()},
		log:                    mgr.GetLogger(),
		statusHandler:          status.NewDefaultStatusHandler(),
		reconciliationInterval: reconciliationInterval,
	}
}

func (r *IstioReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log.Info("Was called to reconcile Kyma Istio Service Mesh")

	istioCR := operatorv1alpha1.Istio{}
	if err := r.Client.Get(ctx, req.NamespacedName, &istioCR); err != nil {
		if apierrors.IsNotFound(err) {
			r.log.Info("Skipped reconciliation, because Istio CR was not found", "request object", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		r.log.Error(err, "Error during fetching Istio CR")
		return r.statusHandler.SetError(ctx, described_errors.NewDescribedError(err, "Could not get Istio CR"), r.Client, &istioCR, metav1.Condition{}, ErrorRetryTime)
	}

	if istioCR.GetNamespace() != namespace {
		errWrongNS := errors.New(fmt.Sprintf("Istio CR is not in %s namespace", namespace))
		r.log.Error(errWrongNS, "Skipped reconciliation")
		return r.statusHandler.SetError(ctx, described_errors.NewDescribedError(errWrongNS, "Error occurred during reconciliation of Istio CR"), r.Client, &istioCR, metav1.Condition{}, ErrorRetryTime)
	}

	istioCR, err := r.istioInstallation.Reconcile(ctx, istioCR, IstioResourceListDefaultPath)
	if err != nil {
		r.log.Error(err, "Error occurred during reconciliation of Istio installation")
		return r.statusHandler.SetError(ctx, err, r.Client, &istioCR, metav1.Condition{}, ErrorRetryTime)
	}

	// If there are no finalizers left, we must assume that the resource is deleted and therefore must stop the reconciliation
	// to prevent accidental read or write to the resource.
	if !istioCR.HasFinalizer() {
		r.log.Info("Finish reconciliation as all finalizers have been removed")
		return ctrl.Result{}, nil
	}

	// We do not want to safeguard the Istio sidecar reconciliation by checking whether Istio has to be installed. The
	// reason for this is that we want to guarantee the restart of the proxies during the next reconciliation even if an
	// error occurs in the reconciliation of the Istio upgrade after the Istio upgrade.
	proxyErr := r.proxySidecars.Reconcile(ctx, istioCR)
	if proxyErr != nil {
		r.log.Error(proxyErr, "Error occurred during reconciliation of Istio Sidecars")
		return r.statusHandler.SetError(ctx, described_errors.NewDescribedError(proxyErr, "Error occurred during reconciliation of Istio Sidecars"), r.Client, &istioCR, metav1.Condition{}, ErrorRetryTime)
	}

	// Put applied configuration in annotation
	istioCR, updateErr := istio.UpdateLastAppliedConfiguration(istioCR, IstioTag)
	if updateErr != nil {
		r.log.Error(updateErr, "Error updating LastAppliedConfiguration")
		return r.statusHandler.SetError(ctx, described_errors.NewDescribedError(updateErr, "Error updating LastAppliedConfiguration"), r.Client, &istioCR, metav1.Condition{}, ErrorRetryTime)
	}

	updateErr = r.Client.Update(ctx, &istioCR)
	if updateErr != nil {
		r.log.Error(updateErr, "Error during update of IstioCR")
		return r.statusHandler.SetError(ctx, described_errors.NewDescribedError(updateErr, "Error during update of IstioCR"), r.Client, &istioCR, metav1.Condition{}, ErrorRetryTime)
	}

	r.log.Info("Reconcile completed")

	return r.statusHandler.SetReady(ctx, r.Client, &istioCR, metav1.Condition{}, r.reconciliationInterval)
}

// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;create;update;patch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list
func (r *IstioReconciler) SetupWithManager(mgr ctrl.Manager, rateLimiter RateLimiter) error {
	r.Config = mgr.GetConfig()

	if err := mgr.GetFieldIndexer().IndexField(context.TODO(), &corev1.Pod{}, "status.phase", func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		return []string{string(pod.Status.Phase)}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Istio{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
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

// TemplateRateLimiter implements a rate limiter for a client-go.workqueue.  It has
// both an overall (token bucket) and per-item (exponential) rate limiting.
func TemplateRateLimiter(failureBaseDelay time.Duration, failureMaxDelay time.Duration,
	frequency int, burst int,
) ratelimiter.RateLimiter {
	return workqueue.NewMaxOfRateLimiter(
		workqueue.NewItemExponentialFailureRateLimiter(failureBaseDelay, failureMaxDelay),
		&workqueue.BucketRateLimiter{Limiter: rate.NewLimiter(rate.Limit(frequency), burst)})
}
