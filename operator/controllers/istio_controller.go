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

	"golang.org/x/time/rate"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/ratelimiter"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/reconciliations/istio"
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
		log:               mgr.GetLogger(),
	}
}

func (r *IstioReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log.Info("Was called to reconcile Kyma Istio Service Mesh")

	istioCR := operatorv1alpha1.Istio{}
	if err := r.Client.Get(ctx, req.NamespacedName, &istioCR); err != nil {
		if errors.IsNotFound(err) {
			r.UpdateStatus(ctx, &istioCR, operatorv1alpha1.Error, metav1.Condition{})
		}
		r.log.Error(err, "Error during fetching Istio CR")
		return r.UpdateStatus(ctx, &istioCR, operatorv1alpha1.Error, metav1.Condition{})
	}

	istioTag := fmt.Sprintf("%s-%s", IstioVersion, IstioImageBase)

	// Evaluate what changed since last reconciliation
	reconciliationTrigger, err := istio.EvaluateIstioCRChanges(istioCR, istioTag)
	if err != nil {
		r.log.Error(err, "Error evaluating IstioCR changes")
		return r.UpdateStatus(ctx, &istioCR, operatorv1alpha1.Error, metav1.Condition{})
	}

	// Perform Istio installation only when needed
	if reconciliationTrigger.NeedsIstioInstall() {
		// Update status to Processing when install is in progress
		res, err := r.UpdateStatus(ctx, &istioCR, operatorv1alpha1.Processing, metav1.Condition{})
		if err != nil {
			return res, err
		}

		err = r.istioInstallation.Reconcile(ctx, &istioCR, r.Client)
		if err != nil {
			r.log.Error(err, "Error occurred during reconciliation of Istio Operator")
			return r.UpdateStatus(ctx, &istioCR, operatorv1alpha1.Error, metav1.Condition{})
		}
	} else {
		ctrl.Log.Info("Install of Istio was skipped")
	}

	// Put applied configuration in annotation
	istioCR, err = istio.UpdateLastAppliedConfiguration(istioCR, istioTag)
	if err != nil {
		r.log.Error(err, "Error updating LastAppliedConfiguration")
		return r.UpdateStatus(ctx, &istioCR, operatorv1alpha1.Error, metav1.Condition{})
	}

	err = r.Client.Update(ctx, &istioCR)
	if err != nil {
		r.log.Error(err, "Error during update of IstioCR")
		return r.UpdateStatus(ctx, &istioCR, operatorv1alpha1.Error, metav1.Condition{})
	}

	// Update status to Ready
	return r.UpdateStatus(ctx, &istioCR, operatorv1alpha1.Ready, metav1.Condition{})
}

// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;create;update;patch
func (r *IstioReconciler) SetupWithManager(mgr ctrl.Manager, rateLimiter RateLimiter) error {
	r.Config = mgr.GetConfig()

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

func (r *IstioReconciler) UpdateStatus(ctx context.Context, istioCR *operatorv1alpha1.Istio, state operatorv1alpha1.State, condition metav1.Condition) (ctrl.Result, error) {
	istioCR.Status.State = state
	meta.SetStatusCondition(istioCR.Status.Conditions, condition)

	if err := r.Client.Status().Update(ctx, istioCR); err != nil {
		r.log.Error(err, "Unable to update the status")
		return ctrl.Result{
			RequeueAfter: time.Minute * 5,
		}, err
	}

	return ctrl.Result{}, nil
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
