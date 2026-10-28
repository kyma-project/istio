package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	"github.com/kyma-project/istio/operator/restarter/internal/restart"
)

// workloadStrategy captures the per-type differences between the Deployment,
// StatefulSet, DaemonSet, and ReplicaSet reconcilers.
type workloadStrategy[T client.Object] struct {
	logKey string
	newObj func() T
	// template allows access to the Pod template in the workload spec.
	// Example: for Deployment return `spec.template`.
	template func(T) *corev1.PodTemplateSpec
	// listPods returns the pods the evaluators run against for this workload. A
	// nil list with a nil error signals an early return with no evaluation (e.g.
	// a Deployment with no usable selector).
	listPods func(context.Context, client.Client, ctrl.Request, T) (*corev1.PodList, error)
	// rolledOut reports whether a prior rollout has fully settled
	rolledOut func(T) bool
	// restart performs the kind-specific restart (template patch, or pod delete
	// for ownerless ReplicaSets) and stamps the restart timestamp and fingerprint.
	restart func(context.Context, client.Client, T, []*corev1.Pod, string) error
}

func reconcileWorkload[T client.Object](
	ctx context.Context,
	c client.Client,
	evaluators evaluation.Source,
	recorder record.EventRecorder,
	req ctrl.Request,
	s workloadStrategy[T],
) (ctrl.Result, error) {
	l := logf.FromContext(ctx).WithValues(s.logKey, req.NamespacedName)
	l.V(1).Info("Reconciling workload")

	obj := s.newObj()
	if err := c.Get(ctx, req.NamespacedName, obj); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ns := &corev1.Namespace{}
	if err := c.Get(ctx, client.ObjectKey{Name: req.Namespace}, ns); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	tmpl := s.template(obj)
	if !evaluation.ShouldInject(ns.Labels, tmpl.Labels, tmpl.Annotations) {
		l.V(1).Info("")
		return ctrl.Result{}, nil
	}

	podList, err := s.listPods(ctx, c, req, obj)
	if err != nil {
		l.Error(err, "Failed to list Pods")
		return ctrl.Result{}, err
	}
	if podList == nil {
		return ctrl.Result{}, nil
	}

	pods := make([]*corev1.Pod, len(podList.Items))
	for i := range podList.Items {
		pods[i] = &podList.Items[i]
	}

	evs, fingerprint, err := evaluators.Snapshot(ctx)
	if err != nil {
		l.Error(err, "Failed to snapshot evaluators")
		return ctrl.Result{}, err
	}
	decision, kind := evaluation.EvaluateDetailed(ctx, obj, pods, evs)
	l.V(1).Info("Evaluation decision", "decision", decision, "pods", len(pods))
	emitDecisionEvent(recorder, obj, decision)

	if decision == evaluation.Restart {
		if kind == evaluation.KindConfigDrift && restart.AlreadyRestartedForConfig(tmpl.Annotations[restart.HashAnnotation], fingerprint) {
			l.V(1).Info("Workload already restarted for current config; skipping", "fingerprint", fingerprint)
			return ctrl.Result{}, nil
		}
		if !s.rolledOut(obj) {
			l.V(1).Info("Rollout in progress; deferring restart")
			return ctrl.Result{}, nil
		}
		l.V(1).Info("Restarting workload", "fingerprint", fingerprint)
		if err := s.restart(ctx, c, obj, pods, fingerprint); err != nil {
			l.Error(err, "Failed to restart workload")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// listPodsByOwnerIndex returns a listPods function that selects pods via a
// field index keyed on the owning workload's UID.
func listPodsByOwnerIndex[T client.Object](ownerKey string) func(context.Context, client.Client, ctrl.Request, T) (*corev1.PodList, error) {
	return func(ctx context.Context, c client.Client, req ctrl.Request, obj T) (*corev1.PodList, error) {
		podList := &corev1.PodList{}
		if err := c.List(ctx, podList,
			client.InNamespace(req.Namespace),
			client.MatchingFields{ownerKey: string(obj.GetUID())},
		); err != nil {
			return nil, err
		}
		return podList, nil
	}
}

func podOwnerIndexer(kind string) func(client.Object) []string {
	return func(obj client.Object) []string {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil
		}
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == kind && ref.APIVersion == "apps/v1" {
				return []string{string(ref.UID)}
			}
		}
		return nil
	}
}

func objectListToClientObjects[ListPtr client.ObjectList, ItemPtr interface {
	*Item
	client.Object
}, Item any](itemsOf func(ListPtr) []Item) func(client.ObjectList) []client.Object {
	return func(list client.ObjectList) []client.Object {
		items := itemsOf(list.(ListPtr))
		objects := make([]client.Object, len(items))
		for i := range items {
			objects[i] = ItemPtr(&items[i])
		}
		return objects
	}
}
