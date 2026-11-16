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
	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	"github.com/kyma-project/istio/operator/restarter/internal/restart"
)

const replicaSetPodOwnerKey = ".metadata.replicaSetOwner"

// ReplicaSetReconciler reconciles a ReplicaSet object
type ReplicaSetReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Evaluators evaluation.Source
	Recorder   record.EventRecorder
}

// +kubebuilder:rbac:groups=apps,resources=replicasets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=replicasets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=replicasets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *ReplicaSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, r.Evaluators, r.Recorder, req, replicaSetStrategy{})
}

type replicaSetStrategy struct{}

func (replicaSetStrategy) logKey() string { return "replicaset" }

func (replicaSetStrategy) newObj() *appsv1.ReplicaSet { return &appsv1.ReplicaSet{} }

func (replicaSetStrategy) template(rs *appsv1.ReplicaSet) *corev1.PodTemplateSpec {
	return &rs.Spec.Template
}

func (replicaSetStrategy) listPods(ctx context.Context, c client.Client, req ctrl.Request, rs *appsv1.ReplicaSet) (*corev1.PodList, error) {
	return listPodsByOwnerIndex[*appsv1.ReplicaSet](replicaSetPodOwnerKey)(ctx, c, req, rs)
}

func (replicaSetStrategy) rolledOut(rs *appsv1.ReplicaSet) bool {
	return restart.ReplicaSetRolledOut(rs)
}

func (replicaSetStrategy) restart(ctx context.Context, c client.Client, rs *appsv1.ReplicaSet, pods []*corev1.Pod, fp string) error {
	return restart.ReplicaSetPods(ctx, c, rs, pods, fp)
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
				objectListToClientObjects(func(l *appsv1.ReplicaSetList) []appsv1.ReplicaSet { return l.Items }),
				isOwnerless,
			),
		), builder.WithPredicates(LastAppliedConfigurationChangedPredicate{})).
		Named("replicaset").
		Complete(r)
}

func isOwnerless(obj client.Object) bool {
	return len(obj.GetOwnerReferences()) == 0
}

func ownerlessReplicaSet() predicate.Predicate {
	return predicate.NewPredicateFuncs(isOwnerless)
}
