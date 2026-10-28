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

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/restarter/internal/evaluation"
	"github.com/kyma-project/istio/restarter/internal/restart"
)

const statefulSetPodOwnerKey = ".metadata.statefulSetOwner"

// StatefulSetReconciler reconciles a StatefulSet object
type StatefulSetReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Evaluators evaluation.Source
	Recorder   record.EventRecorder
}

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=statefulsets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, r.Evaluators, r.Recorder, req, workloadStrategy[*appsv1.StatefulSet]{
		logKey:    "statefulset",
		newObj:    func() *appsv1.StatefulSet { return &appsv1.StatefulSet{} },
		template:  func(s *appsv1.StatefulSet) *corev1.PodTemplateSpec { return &s.Spec.Template },
		listPods:  listPodsByOwnerIndex[*appsv1.StatefulSet](statefulSetPodOwnerKey),
		rolledOut: restart.StatefulSetRolledOut,
		restart: func(ctx context.Context, c client.Client, s *appsv1.StatefulSet, _ []*corev1.Pod, fp string) error {
			return restart.StatefulSet(ctx, c, s, fp)
		},
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *StatefulSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("statefulset-controller")

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&corev1.Pod{},
		statefulSetPodOwnerKey,
		podOwnerIndexer("StatefulSet"),
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		Watches(&operatorv1alpha2.Istio{}, handler.EnqueueRequestsFromMapFunc(
			enqueueWorkloadsOnIstioChange(r.Client,
				func() client.ObjectList { return &appsv1.StatefulSetList{} },
				objectListToClientObjects(func(l *appsv1.StatefulSetList) []appsv1.StatefulSet { return l.Items }),
				nil,
			),
		), builder.WithPredicates(LastAppliedConfigurationChangedPredicate{})).
		Named("statefulset").
		Complete(r)
}
