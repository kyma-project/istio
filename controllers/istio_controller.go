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
	"github.com/pkg/errors"
	"time"

	"github.com/kyma-project/istio/operator/internal/filter"

	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/reconciliations/ingress_gateway"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	"k8s.io/client-go/util/retry"

	"github.com/kyma-project/istio/operator/internal/manifest"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"github.com/kyma-project/istio/operator/internal/reconciliations/proxy"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	IstioVersion                 = "1.19.0"
	IstioImageBase               = "distroless"
	IstioResourceListDefaultPath = "manifests/controlled_resources_list.yaml"
)

var IstioTag = fmt.Sprintf("%s-%s", IstioVersion, IstioImageBase)

func NewReconciler(mgr manager.Manager, reconciliationInterval time.Duration) *IstioReconciler {
	merger := manifest.NewDefaultIstioMerger()

	efReferer := istio_resources.NewEnvoyFilterAllowPartialReferer(mgr.GetClient())

	istioResources := []istio_resources.Resource{efReferer}
	istioResources = append(istioResources, istio_resources.NewGatewayKyma(mgr.GetClient()))
	istioResources = append(istioResources, istio_resources.NewVirtualServiceHealthz(mgr.GetClient()))
	istioResources = append(istioResources, istio_resources.NewPeerAuthenticationMtls(mgr.GetClient()))
	istioResources = append(istioResources, istio_resources.NewConfigMapControlPlane(mgr.GetClient()))
	istioResources = append(istioResources, istio_resources.NewConfigMapMesh(mgr.GetClient()))
	istioResources = append(istioResources, istio_resources.NewConfigMapPerformance(mgr.GetClient()))
	istioResources = append(istioResources, istio_resources.NewConfigMapService(mgr.GetClient()))
	istioResources = append(istioResources, istio_resources.NewConfigMapWorkload(mgr.GetClient()))

	return &IstioReconciler{
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
		istioInstallation:      &istio.Installation{Client: mgr.GetClient(), IstioClient: istio.NewIstioClient(), IstioVersion: IstioVersion, IstioImageBase: IstioImageBase, Merger: &merger},
		proxySidecars:          &proxy.Sidecars{IstioVersion: IstioVersion, IstioImageBase: IstioImageBase, Log: mgr.GetLogger(), Client: mgr.GetClient(), Merger: &merger, Predicates: []filter.SidecarProxyPredicate{efReferer}},
		istioResources:         istio_resources.NewReconciler(mgr.GetClient(), istioResources),
		ingressGateway:         ingress_gateway.NewReconciler(mgr.GetClient(), []filter.IngressGatewayPredicate{efReferer}),
		log:                    mgr.GetLogger(),
		statusHandler:          newStatusHandler(mgr.GetClient()),
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
		r.log.Error(err, "Could not get Istio CR")
		return ctrl.Result{}, err
	}

	if istioCR.GetNamespace() != namespace {
		errWrongNS := fmt.Errorf("istio CR is not in %s namespace", namespace)
		return r.terminateReconciliation(ctx, istioCR, described_errors.NewDescribedError(errWrongNS, "Stopped Istio CR reconciliation"))
	}

	existingIstioCRs := &operatorv1alpha1.IstioList{}
	if err := r.List(ctx, existingIstioCRs, client.InNamespace(namespace)); err != nil {
		return r.requeueReconciliation(ctx, istioCR, described_errors.NewDescribedError(err, "Unable to list Istio CRs"))
	}

	if len(existingIstioCRs.Items) > 1 {
		oldestCr := r.getOldestCR(existingIstioCRs)
		if istioCR.GetUID() != oldestCr.GetUID() {
			errNotOldestCR := fmt.Errorf("only Istio CR %s in %s reconciles the module", oldestCr.GetName(), oldestCr.GetNamespace())
			return r.terminateReconciliation(ctx, istioCR, described_errors.NewDescribedError(errNotOldestCR, "Stopped Istio CR reconciliation"))
		}
	}

	if istioCR.DeletionTimestamp.IsZero() {
		if err := r.statusHandler.updateToProcessing(ctx, "Reconciling Istio resources", &istioCR); err != nil {
			r.log.Error(err, "Update status to processing failed")
			// We don't update the status to error, because the status update already failed and to avoid another status update error we simply requeue the request.
			return ctrl.Result{}, err
		}
	} else {
		if err := r.statusHandler.updateToDeleting(ctx, &istioCR); err != nil {
			r.log.Error(err, "Update status to deleting failed")
			// We don't update the status to error, because the status update already failed and to avoid another status update error we simply requeue the request.
			return ctrl.Result{}, err
		}
	}

	istioCR, installationErr := r.istioInstallation.Reconcile(ctx, istioCR, IstioResourceListDefaultPath)
	if installationErr != nil {
		return r.requeueReconciliation(ctx, istioCR, installationErr)
	}

	// If there are no finalizers left, we must assume that the resource is deleted and therefore must stop the reconciliation
	// to prevent accidental read or write to the resource.
	if !istioCR.HasFinalizer() {
		r.log.Info("End reconciliation because all finalizers have been removed")
		return ctrl.Result{}, nil
	}

	resourcesErr := r.istioResources.Reconcile(ctx, istioCR)
	if resourcesErr != nil {
		return r.requeueReconciliation(ctx, istioCR, resourcesErr)
	}

	// We do not want to safeguard the Istio sidecar reconciliation by checking whether Istio has to be installed. The
	// reason for this is that we want to guarantee the restart of the proxies during the next reconciliation even if an
	// error occurs in the reconciliation of the Istio upgrade after the Istio upgrade.
	warningHappened, proxyErr := r.proxySidecars.Reconcile(ctx, istioCR)
	if proxyErr != nil {
		describedErr := described_errors.NewDescribedError(proxyErr, "Error occurred during reconciliation of Istio Sidecars")
		return r.requeueReconciliation(ctx, istioCR, describedErr)
	} else if warningHappened {
		warning := described_errors.NewDescribedError(errors.New("Istio controller could not restart one or more istio-injected pods."), "Please take a look at kyma-system/istio-controller-manager logs to see more information about the warning").SetWarning()
		return r.requeueReconciliation(ctx, istioCR, warning)
	}

	ingressGatewayErr := r.ingressGateway.Reconcile(ctx)
	if ingressGatewayErr != nil {
		return r.requeueReconciliation(ctx, istioCR, ingressGatewayErr)
	}

	return r.finishReconcile(ctx, istioCR, IstioTag)
}

// requeueReconciliation cancels the reconciliation and requeues the request.
func (r *IstioReconciler) requeueReconciliation(ctx context.Context, istioCR operatorv1alpha1.Istio, err described_errors.DescribedError) (ctrl.Result, error) {
	statusUpdateErr := r.statusHandler.updateToError(ctx, err, &istioCR)
	if statusUpdateErr != nil {
		r.log.Error(statusUpdateErr, "Error during updating status to error")
	}

	r.log.Error(err, "Reconcile failed")
	return ctrl.Result{}, err
}

// terminateReconciliation stops the reconciliation and does not requeue the request.
func (r *IstioReconciler) terminateReconciliation(ctx context.Context, istioCR operatorv1alpha1.Istio, err described_errors.DescribedError) (ctrl.Result, error) {
	statusUpdateErr := r.statusHandler.updateToError(ctx, err, &istioCR)
	if statusUpdateErr != nil {
		r.log.Error(statusUpdateErr, "Error during updating status to error")
		// In case the update of the status fails we must requeue the request, because otherwise the Error state is never visible in the CR.
		return ctrl.Result{}, statusUpdateErr
	}

	r.log.Error(err, "Reconcile failed, but won't requeue")
	return ctrl.Result{}, nil
}

func (r *IstioReconciler) finishReconcile(ctx context.Context, istioCR operatorv1alpha1.Istio, istioTag string) (ctrl.Result, error) {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if getErr := r.Client.Get(ctx, client.ObjectKeyFromObject(&istioCR), &istioCR); getErr != nil {
			return getErr
		}

		istioCR, lastAppliedErr := istio.UpdateLastAppliedConfiguration(istioCR, istioTag)
		if lastAppliedErr != nil {
			return lastAppliedErr
		}

		return r.Client.Update(ctx, &istioCR)
	})

	if retryErr != nil {
		describedErr := described_errors.NewDescribedError(retryErr, "Error updating LastAppliedConfiguration")
		return r.requeueReconciliation(ctx, istioCR, describedErr)
	}

	if statusErr := r.statusHandler.updateToReady(ctx, &istioCR); statusErr != nil {
		r.log.Error(statusErr, "Error during updating status to ready")
		return ctrl.Result{}, statusErr
	}

	r.log.Info("Reconcile finished")
	return ctrl.Result{RequeueAfter: r.reconciliationInterval}, nil
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

func (r *IstioReconciler) getOldestCR(istioCRs *operatorv1alpha1.IstioList) *operatorv1alpha1.Istio {
	oldest := istioCRs.Items[0]
	for _, item := range istioCRs.Items {
		timestamp := &item.CreationTimestamp
		if !(oldest.CreationTimestamp.Before(timestamp)) {
			oldest = item
		}
	}
	return &oldest
}
