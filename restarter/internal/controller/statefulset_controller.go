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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

const statefulSetPodOwnerKey = ".metadata.statefulSetOwner"

// StatefulSetReconciler reconciles a StatefulSet object
type StatefulSetReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Evaluators evaluation.Source
	Recorder   record.EventRecorder
	IstioCRKey types.NamespacedName
}

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := logf.FromContext(ctx).WithValues("statefulset", req.NamespacedName)
	l.V(1).Info("Reconciling StatefulSet")

	sts := &appsv1.StatefulSet{}
	if err := r.Get(ctx, req.NamespacedName, sts); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: req.Namespace}, ns); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !evaluation.ShouldInject(ns.Labels, sts.Spec.Template.Labels, sts.Spec.Template.Annotations) {
		l.V(1).Info("Injection is disabled for this workload. Skipping evaluation.")
		return ctrl.Result{}, nil
	}

	podList := &corev1.PodList{}
	if err := r.List(ctx, podList,
		client.InNamespace(req.Namespace),
		client.MatchingFields{statefulSetPodOwnerKey: string(sts.UID)},
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
	decision, kind := evaluation.EvaluateDetailed(ctx, sts, pods, evs)
	l.V(1).Info("Evaluation decision", "decision", decision, "pods", len(pods))
	emitDecisionEvent(r.Recorder, sts, decision)

	if decision == evaluation.Restart {
		if kind == evaluation.KindConfigDrift && alreadyRestartedForConfig(sts.Spec.Template.Annotations[FingerprintAnnotation], fingerprint) {
			l.V(1).Info("Workload already restarted for current config; skipping", "fingerprint", fingerprint)
			return ctrl.Result{}, nil
		}
		if !statefulSetRolledOut(sts) {
			l.V(1).Info("Rollout in progress; deferring restart")
			return ctrl.Result{}, nil
		}
		l.V(1).Info("Restarting StatefulSet", "fingerprint", fingerprint)
		if err := restartStatefulSet(ctx, r.Client, sts, fingerprint); err != nil {
			l.Error(err, "Failed to restart StatefulSet")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StatefulSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("statefulset-controller")

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&corev1.Pod{},
		statefulSetPodOwnerKey,
		func(obj client.Object) []string {
			pod, ok := obj.(*corev1.Pod)
			if !ok {
				return nil
			}
			for _, ref := range pod.OwnerReferences {
				if ref.Kind == "StatefulSet" && ref.APIVersion == "apps/v1" {
					return []string{string(ref.UID)}
				}
			}
			return nil
		},
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		Watches(&operatorv1alpha2.Istio{}, handler.EnqueueRequestsFromMapFunc(
			enqueueWorkloadsOnIstioChange(r.Client,
				func() client.ObjectList { return &appsv1.StatefulSetList{} },
				func(list client.ObjectList) []client.Object {
					items := list.(*appsv1.StatefulSetList).Items
					objs := make([]client.Object, len(items))
					for i := range items {
						objs[i] = &items[i]
					}
					return objs
				},
				nil,
			),
		)).
		Named("statefulset").
		Complete(r)
}
