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
	"github.com/kyma-project/istio/operator/internal/restarter"
	"github.com/kyma-project/istio/operator/internal/validation"
	"net/http"
	"time"

	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars"

	"github.com/kyma-project/istio/operator/internal/described_errors"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio_resources"
	"github.com/kyma-project/istio/operator/internal/status"
	"k8s.io/client-go/util/retry"

	"github.com/kyma-project/istio/operator/internal/manifest"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
	"golang.org/x/time/rate"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
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
	IstioVersion                 = "1.20.3"
	IstioImageBase               = "distroless"
	IstioResourceListDefaultPath = "manifests/controlled_resources_list.yaml"
)

var IstioTag = fmt.Sprintf("%s-%s", IstioVersion, IstioImageBase)

func NewReconciler(mgr manager.Manager, reconciliationInterval time.Duration) *IstioReconciler {
	merger := manifest.NewDefaultIstioMerger()

	efReferer := istio_resources.NewEnvoyFilterAllowPartialReferer(mgr.GetClient())
	statusHandler := status.NewStatusHandler(mgr.GetClient())
	restarters := []Restarter{
		// inject status handlers here
		restarter.NewIngressGatewayRestarter(mgr.GetClient(), []filter.IngressGatewayPredicate{efReferer}, statusHandler),
		restarter.NewSidecarsRestarter(IstioVersion, IstioImageBase, mgr.GetLogger(), mgr.GetClient(), &merger, sidecars.NewProxyResetter(), []filter.SidecarProxyPredicate{efReferer}, statusHandler),
	}

	return &IstioReconciler{
		Client:                 mgr.GetClient(),
		Scheme:                 mgr.GetScheme(),
		istioInstallation:      &istio.Installation{Client: mgr.GetClient(), IstioClient: istio.NewIstioClient(), IstioVersion: IstioVersion, IstioImageBase: IstioImageBase, Merger: &merger},
		istioResources:         istio_resources.NewReconciler(mgr.GetClient(), clusterconfig.NewHyperscalerClient(&http.Client{Timeout: 1 * time.Second})),
		restarters:             restarters,
		log:                    mgr.GetLogger(),
		statusHandler:          statusHandler,
		reconciliationInterval: reconciliationInterval,
	}
}

func (r *IstioReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log.Info("Was called to reconcile Kyma Istio Service Mesh")

	istioCR := operatorv1alpha2.Istio{}
	if err := r.Client.Get(ctx, req.NamespacedName, &istioCR); err != nil {
		if apierrors.IsNotFound(err) {
			r.log.Info("Skipped reconciliation, because Istio CR was not found", "request object", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		r.log.Error(err, "Could not get Istio CR")
		return ctrl.Result{}, err
	}

	err := validation.ValidateAuthorizers(istioCR)
	if err != nil {
		return r.terminateReconciliation(ctx, &istioCR, err, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonValidationFailed))
	}

	if istioCR.GetNamespace() != namespace {
		errWrongNS := fmt.Errorf("Istio CR is not in %s namespace", namespace)
		return r.terminateReconciliation(ctx, &istioCR, described_errors.NewDescribedError(errWrongNS, "Stopped Istio CR reconciliation"),
			operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonReconcileFailed))
	}

	existingIstioCRs := &operatorv1alpha2.IstioList{}
	if err := r.List(ctx, existingIstioCRs, client.InNamespace(namespace)); err != nil {
		return r.requeueReconciliation(ctx, &istioCR, described_errors.NewDescribedError(err, "Unable to list Istio CRs"),
			operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonReconcileFailed))
	}

	if len(existingIstioCRs.Items) > 1 {
		oldestCr := r.getOldestCR(existingIstioCRs)
		if istioCR.GetUID() != oldestCr.GetUID() {
			errNotOldestCR := fmt.Errorf("only Istio CR %s in %s reconciles the module", oldestCr.GetName(), oldestCr.GetNamespace())
			return r.terminateReconciliation(ctx, &istioCR, described_errors.NewDescribedError(errNotOldestCR, "Stopped Istio CR reconciliation").SetWarning(),
				operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonOlderCRExists))
		}
	}

	if istioCR.DeletionTimestamp.IsZero() {
		if err := r.statusHandler.UpdateToProcessing(ctx, &istioCR); err != nil {
			r.log.Error(err, "Update status to processing failed")
			// We don't update the status to error, because the status update already failed and to avoid another status update error we simply requeue the request.
			return ctrl.Result{}, err
		}
	} else {
		if err := r.statusHandler.UpdateToDeleting(ctx, &istioCR); err != nil {
			r.log.Error(err, "Update status to deleting failed")
			// We don't update the status to error, because the status update already failed and to avoid another status update error we simply requeue the request.
			return ctrl.Result{}, err
		}
	}

	installationErr := r.istioInstallation.Reconcile(ctx, &istioCR, r.statusHandler, IstioResourceListDefaultPath)
	if installationErr != nil {
		return r.requeueReconciliation(ctx, &istioCR, installationErr, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonIstioInstallUninstallFailed))
	}

	// If there are no finalizers left, we must assume that the resource is deleted and therefore must stop the reconciliation
	// to prevent accidental read or write to the resource.
	if !istioCR.HasFinalizers() {
		r.log.Info("End reconciliation because all finalizers have been removed")
		return ctrl.Result{}, nil
	}

	resourcesErr := r.istioResources.Reconcile(ctx, istioCR)
	if resourcesErr != nil {
		return r.requeueReconciliation(ctx, &istioCR, resourcesErr, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonCRsReconcileFailed))
	}

	// We do not want to safeguard the Istio sidecar reconciliation by checking whether Istio has to be installed. The
	// reason for this is that we want to guarantee the restart of the proxies during the next reconciliation even if an
	// error occurs in the reconciliation of the Istio upgrade after the Istio upgrade.
	// should we pass Istio to the restart methods to set conditions?
	var restartersErr []described_errors.DescribedError
	for _, res := range r.restarters {
		restarterErr := res.Restart(ctx, &istioCR)
		if restarterErr != nil {
			restartersErr = append(restartersErr, restarterErr)
		}
	}
	if restartersFailureErr := detectRestartersFailure(restartersErr); restartersFailureErr != nil {
		err := r.statusHandler.UpdateToError(ctx, &istioCR, restartersFailureErr)
		if err != nil {
			r.log.Error(err, "Error during updating status to error")
		}
		return ctrl.Result{}, err
	}
	//if len(restartersErr) > 0 {
	//	var candidate described_errors.DescribedError
	//	for _, err := range restartersErr {
	//		if candidate == nil || err.Level() < candidate.Level() {
	//			candidate = err
	//		}
	//	}
	//
	//	err := r.statusHandler.UpdateToError(ctx, &istioCR, candidate)
	//	if err != nil {
	//		r.log.Error(err, "Error during updating status to error")
	//	}
	//
	//	return ctrl.Result{}, candidate
	//}

	return r.finishReconcile(ctx, &istioCR, IstioTag)
}

func detectRestartersFailure(restartersErr []described_errors.DescribedError) described_errors.DescribedError {
	var candidate described_errors.DescribedError
	for _, err := range restartersErr {
		if candidate == nil || err.Level() < candidate.Level() {
			candidate = err
		}
	}
	return candidate
}

// requeueReconciliation cancels the reconciliation and requeues the request.
func (r *IstioReconciler) requeueReconciliation(ctx context.Context, istioCR *operatorv1alpha2.Istio, err described_errors.DescribedError, reason operatorv1alpha2.ReasonWithMessage) (ctrl.Result, error) {
	if err.ShouldSetCondition() {
		r.setConditionForError(istioCR, reason)
	}
	statusUpdateErr := r.statusHandler.UpdateToError(ctx, istioCR, err)
	if statusUpdateErr != nil {
		r.log.Error(statusUpdateErr, "Error during updating status to error")
	}

	r.log.Error(err, "Reconcile failed")
	return ctrl.Result{}, err
}

func (r *IstioReconciler) setRequeueAndContinue(err described_errors.DescribedError) {
}

// terminateReconciliation stops the reconciliation and does not requeue the request.
func (r *IstioReconciler) terminateReconciliation(ctx context.Context, istioCR *operatorv1alpha2.Istio, err described_errors.DescribedError, reason operatorv1alpha2.ReasonWithMessage) (ctrl.Result, error) {
	if err.ShouldSetCondition() {
		r.setConditionForError(istioCR, reason)
	}
	statusUpdateErr := r.statusHandler.UpdateToError(ctx, istioCR, err)
	if statusUpdateErr != nil {
		r.log.Error(statusUpdateErr, "Error during updating status to error")
		// In case the update of the status fails we must requeue the request, because otherwise the Error state is never visible in the CR.
		return ctrl.Result{}, statusUpdateErr
	}

	r.log.Error(err, "Reconcile failed, but won't requeue")
	return ctrl.Result{}, nil
}

func (r *IstioReconciler) finishReconcile(ctx context.Context, istioCR *operatorv1alpha2.Istio, istioTag string) (ctrl.Result, error) {
	if err := r.updateLastAppliedConfiguration(ctx, client.ObjectKeyFromObject(istioCR), istioTag); err != nil {
		describedErr := described_errors.NewDescribedError(err, "Error updating LastAppliedConfiguration")
		return r.requeueReconciliation(ctx, istioCR, describedErr, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonReconcileFailed))
	}

	r.statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonReconcileSucceeded))
	if err := r.statusHandler.UpdateToReady(ctx, istioCR); err != nil {
		r.log.Error(err, "Error during updating status to ready")
		return ctrl.Result{}, err
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
		For(&operatorv1alpha2.Istio{}).
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

func (r *IstioReconciler) getOldestCR(istioCRs *operatorv1alpha2.IstioList) *operatorv1alpha2.Istio {
	oldest := istioCRs.Items[0]
	for _, item := range istioCRs.Items {
		timestamp := &item.CreationTimestamp
		if !(oldest.CreationTimestamp.Before(timestamp)) {
			oldest = item
		}
	}
	return &oldest
}

func (r *IstioReconciler) updateLastAppliedConfiguration(ctx context.Context, objectKey types.NamespacedName, istioTag string) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		lacIstioCR := operatorv1alpha2.Istio{}
		if err := r.Client.Get(ctx, objectKey, &lacIstioCR); err != nil {
			return err
		}
		lastAppliedErr := istio.UpdateLastAppliedConfiguration(&lacIstioCR, istioTag)
		if lastAppliedErr != nil {
			return lastAppliedErr
		}
		return r.Client.Update(ctx, &lacIstioCR)
	})
}

func (r *IstioReconciler) setConditionForError(istioCR *operatorv1alpha2.Istio, reason operatorv1alpha2.ReasonWithMessage) {
	if !operatorv1alpha2.IsReadyTypeCondition(reason) {
		r.statusHandler.SetCondition(istioCR, operatorv1alpha2.NewReasonWithMessage(operatorv1alpha2.ConditionReasonReconcileFailed))
	}
	r.statusHandler.SetCondition(istioCR, reason)
}
