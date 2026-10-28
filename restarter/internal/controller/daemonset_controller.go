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

const daemonSetPodOwnerKey = ".metadata.daemonSetOwner"

// DaemonSetReconciler reconciles a DaemonSet object
type DaemonSetReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Evaluators evaluation.Source
	Recorder   record.EventRecorder
}

// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=daemonsets/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=events,verbs=create;patch

func (r *DaemonSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	return reconcileWorkload(ctx, r.Client, r.Evaluators, r.Recorder, req, workloadStrategy[*appsv1.DaemonSet]{
		logKey:    "daemonset",
		newObj:    func() *appsv1.DaemonSet { return &appsv1.DaemonSet{} },
		template:  func(d *appsv1.DaemonSet) *corev1.PodTemplateSpec { return &d.Spec.Template },
		listPods:  listPodsByOwnerIndex[*appsv1.DaemonSet](daemonSetPodOwnerKey),
		rolledOut: restart.DaemonSetRolledOut,
		restart: func(ctx context.Context, c client.Client, d *appsv1.DaemonSet, _ []*corev1.Pod, fp string) error {
			return restart.DaemonSet(ctx, c, d, fp)
		},
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *DaemonSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("daemonset-controller")

	if err := mgr.GetFieldIndexer().IndexField(
		context.Background(),
		&corev1.Pod{},
		daemonSetPodOwnerKey,
		podOwnerIndexer("DaemonSet"),
	); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.DaemonSet{}).
		Watches(&operatorv1alpha2.Istio{}, handler.EnqueueRequestsFromMapFunc(
			enqueueWorkloadsOnIstioChange(r.Client,
				func() client.ObjectList { return &appsv1.DaemonSetList{} },
				objectListToClientObjects(func(l *appsv1.DaemonSetList) []appsv1.DaemonSet { return l.Items }),
				nil,
			),
		), builder.WithPredicates(LastAppliedConfigurationChangedPredicate{})).
		Named("daemonset").
		Complete(r)
}
