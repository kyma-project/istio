package controller

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
	"github.com/kyma-project/istio/operator/workload-controller/internal/restart"
)

// workloadStrategy captures the per-type differences between the Deployment,
// StatefulSet, DaemonSet, and ReplicaSet reconcilers. Everything else in the
// reconcile loop — the namespace injection gate, pod evaluation, the config
// fingerprint guard, the rollout guard, and event emission — is identical
// across kinds and lives in reconcileWorkload.
type workloadStrategy[T client.Object] struct {
	// logKey is the structured-log field name for the workload (e.g. "deployment").
	logKey string
	// newObj allocates a zero value of the workload type to Get into.
	newObj func() T
	// template returns the pod template of the fetched workload, whose
	// annotations hold the restart timestamp and config fingerprint.
	template func(T) *corev1.PodTemplateSpec
	// listPods returns the pods the evaluators run against for this workload. A
	// nil list with a nil error signals an early return with no evaluation (e.g.
	// a Deployment with no usable selector).
	listPods func(context.Context, client.Client, ctrl.Request, T) (*corev1.PodList, error)
	// rolledOut reports whether a prior restart has fully settled; a restart is
	// deferred until it has, so mid-rollout old-revision pods are not read as drift.
	rolledOut func(T) bool
	// restart performs the kind-specific restart (template patch, or pod delete
	// for ownerless ReplicaSets) and stamps the restart timestamp and fingerprint.
	restart func(context.Context, client.Client, T, []*corev1.Pod, string) error
}

// reconcileWorkload is the shared reconcile body for every workload kind. The
// per-type behavior is supplied by s; see workloadStrategy. Behavior is
// identical to the hand-written per-type Reconcile methods it replaces.
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
		if kind == evaluation.KindConfigDrift && restart.AlreadyRestartedForConfig(tmpl.Annotations[restart.FingerprintAnnotation], fingerprint) {
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
// field index keyed on the owning workload's UID. The three owner-ref
// controllers (StatefulSet, DaemonSet, ReplicaSet) register such an index in
// SetupWithManager and share this listing strategy.
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

// podOwnerIndexer returns a field-index function that maps a pod to the UID of
// its apps/v1 owner of the given kind. The three owner-ref controllers register
// it against their owner-key so listPodsByOwnerIndex can select a workload's pods.
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

// objectListItems returns the mapper enqueueWorkloadsOnIstioChange needs: it
// type-asserts the listed objects to *L and returns element pointers as
// []client.Object. Replaces the per-type closure every SetupWithManager used to
// spell out by hand. The *L constraint lets the assertion target the concrete
// list type (e.g. *appsv1.DeploymentList) rather than a bare type parameter.
func objectListItems[L any, PL interface {
	*L
	client.ObjectList
}, I any](items func(PL) []I) func(client.ObjectList) []client.Object {
	return func(list client.ObjectList) []client.Object {
		elems := items(list.(PL))
		objs := make([]client.Object, len(elems))
		for i := range elems {
			objs[i] = any(&elems[i]).(client.Object)
		}
		return objs
	}
}
