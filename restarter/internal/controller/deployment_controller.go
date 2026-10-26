/*
Copyright 2026.

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

package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Evaluators evaluation.Source
	Recorder   record.EventRecorder
	IstioCRKey types.NamespacedName
}

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch
// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.kyma-project.io,resources=istios,verbs=get;list;watch

func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx).WithValues("deployment", req.NamespacedName)
	l.V(1).Info("Reconciling Deployment")

	deployment := &appsv1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, deployment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: req.Namespace}, ns); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !evaluation.ShouldInject(ns.Labels, deployment.Spec.Template.Labels, deployment.Spec.Template.Annotations) {
		l.V(1).Info("Injection is disabled for this workload. Skipping evaluation.")
		return ctrl.Result{}, nil
	}

	// A nil selector means the Deployment is not yet usable; an empty selector would
	// otherwise match every pod in the namespace, so skip both.
	if deployment.Spec.Selector == nil ||
		(len(deployment.Spec.Selector.MatchLabels) == 0 && len(deployment.Spec.Selector.MatchExpressions) == 0) {
		return ctrl.Result{}, nil
	}

	sel, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("invalid deployment selector: %w", err)
	}

	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(req.Namespace),
		client.MatchingLabelsSelector{Selector: sel},
	); err != nil {
		l.Error(err, "Failed to list Pods")
		return ctrl.Result{}, err
	}

	pods := make([]*corev1.Pod, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &podList.Items[i]
	}

	evs, fingerprint, err := r.Evaluators.Snapshot(ctx)
	if err != nil {
		l.Error(err, "Failed to snapshot evaluators")
		return ctrl.Result{}, err
	}
	decision, kind := evaluation.EvaluateDetailed(ctx, deployment, pods, evs)
	l.V(1).Info("Evaluation decision", "decision", decision, "pods", len(pods))
	emitDecisionEvent(r.Recorder, deployment, decision)

	if decision == evaluation.Restart {
		if kind == evaluation.KindConfigDrift && alreadyRestartedForConfig(deployment.Spec.Template.Annotations[FingerprintAnnotation], fingerprint) {
			l.V(1).Info("Workload already restarted for current config; skipping", "fingerprint", fingerprint)
			return ctrl.Result{}, nil
		}
		if !deploymentRolledOut(deployment) {
			l.V(1).Info("Rollout in progress; deferring restart")
			return ctrl.Result{}, nil
		}
		l.V(1).Info("Restarting Deployment", "fingerprint", fingerprint)
		if err := restartDeployment(ctx, r.Client, deployment, fingerprint); err != nil {
			l.Error(err, "Failed to restart Deployment")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("deployment-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Watches(&operatorv1alpha2.Istio{}, handler.EnqueueRequestsFromMapFunc(
			enqueueWorkloadsOnIstioChange(r.Client,
				func() client.ObjectList { return &appsv1.DeploymentList{} },
				func(list client.ObjectList) []client.Object {
					items := list.(*appsv1.DeploymentList).Items
					objs := make([]client.Object, len(items))
					for i := range items {
						objs[i] = &items[i]
					}
					return objs
				},
				nil,
			),
		)).
		Named("deployment").
		Complete(r)
}
