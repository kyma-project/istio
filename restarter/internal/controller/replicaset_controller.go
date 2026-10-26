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
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
)

const replicaSetPodOwnerKey = ".metadata.replicaSetOwner"

// ReplicaSetReconciler reconciles a ReplicaSet object
type ReplicaSetReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Evaluators evaluation.Source
	Recorder   record.EventRecorder
}

// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=replicasets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=replicasets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *ReplicaSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, r.Evaluators, r.Recorder, req, workloadStrategy[*appsv1.ReplicaSet]{
		logKey:    "replicaset",
		newObj:    func() *appsv1.ReplicaSet { return &appsv1.ReplicaSet{} },
		template:  func(rs *appsv1.ReplicaSet) *corev1.PodTemplateSpec { return &rs.Spec.Template },
		listPods:  listPodsByOwnerIndex[*appsv1.ReplicaSet](replicaSetPodOwnerKey),
		rolledOut: replicaSetRolledOut,
		restart:   restartReplicaSetPods,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReplicaSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("replicaset-controller")

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&corev1.Pod{},
		replicaSetPodOwnerKey,
		podOwnerIndexer("ReplicaSet"),
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.ReplicaSet{}, builder.WithPredicates(ownerlessReplicaSet())).
		Watches(&operatorv1alpha2.Istio{}, handler.EnqueueRequestsFromMapFunc(
			enqueueWorkloadsOnIstioChange(r.Client,
				func() client.ObjectList { return &appsv1.ReplicaSetList{} },
				objectListItems(func(l *appsv1.ReplicaSetList) []appsv1.ReplicaSet { return l.Items }),
				isOwnerless,
			),
		)).
		Named("replicaset").
		Complete(r)
}

// isOwnerless reports whether obj has no owner references. ReplicaSets owned by
// a higher-level controller (e.g. Deployment) have their pods evaluated by the
// owner's reconciler; this controller handles only ownerless ones.
func isOwnerless(obj client.Object) bool {
	return len(obj.GetOwnerReferences()) == 0
}

// ownerlessReplicaSet keeps only ReplicaSets with no owner. ReplicaSets owned
// by a higher-level controller (e.g. Deployment) have their pods evaluated by
// the owner's reconciler; enqueueing them here would double-count evaluations
// and emit duplicate events.
func ownerlessReplicaSet() predicate.Predicate {
	return predicate.NewPredicateFuncs(isOwnerless)
}
