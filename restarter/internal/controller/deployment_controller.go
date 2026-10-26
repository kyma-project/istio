package controller

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
	"github.com/kyma-project/istio/operator/workload-controller/internal/restart"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Evaluators evaluation.Source
	Recorder   record.EventRecorder
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
	return reconcileWorkload(ctx, r.Client, r.Evaluators, r.Recorder, req, workloadStrategy[*appsv1.Deployment]{
		logKey:   "deployment",
		newObj:   func() *appsv1.Deployment { return &appsv1.Deployment{} },
		template: func(d *appsv1.Deployment) *corev1.PodTemplateSpec { return &d.Spec.Template },
		listPods: func(ctx context.Context, c client.Client, req ctrl.Request, d *appsv1.Deployment) (*corev1.PodList, error) {
			// A nil selector means the Deployment is not yet usable; an empty selector
			// would otherwise match every pod in the namespace, so skip both by
			// returning a nil list (no evaluation, no error).
			if d.Spec.Selector == nil ||
				(len(d.Spec.Selector.MatchLabels) == 0 && len(d.Spec.Selector.MatchExpressions) == 0) {
				return nil, nil
			}
			sel, err := metav1.LabelSelectorAsSelector(d.Spec.Selector)
			if err != nil {
				return nil, fmt.Errorf("invalid deployment selector: %w", err)
			}
			podList := &corev1.PodList{}
			if err := c.List(ctx, podList,
				client.InNamespace(req.Namespace),
				client.MatchingLabelsSelector{Selector: sel},
			); err != nil {
				return nil, err
			}
			return podList, nil
		},
		rolledOut: restart.DeploymentRolledOut,
		restart: func(ctx context.Context, c client.Client, d *appsv1.Deployment, _ []*corev1.Pod, fp string) error {
			return restart.Deployment(ctx, c, d, fp)
		},
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("deployment-controller")

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Watches(&operatorv1alpha2.Istio{}, handler.EnqueueRequestsFromMapFunc(
			enqueueWorkloadsOnIstioChange(r.Client,
				func() client.ObjectList { return &appsv1.DeploymentList{} },
				objectListItems(func(l *appsv1.DeploymentList) []appsv1.Deployment { return l.Items }),
				nil,
			),
		)).
		Named("deployment").
		Complete(r)
}
