package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
)

// fakeSource is a shared evaluation.Source test helper that returns a static
// evaluator slice, fingerprint, and optional error across controller tests in
// this package.
type fakeSource struct {
	evs []evaluation.Evaluator
	fp  string
	err error
}

func (f fakeSource) Snapshot(_ context.Context) ([]evaluation.Evaluator, string, error) {
	return f.evs, f.fp, f.err
}

func ownerPod(name, kind, apiVersion string, ownerUID types.UID) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Annotations: map[string]string{
				"sidecar.istio.io/status": "injected",
			},
			OwnerReferences: []metav1.OwnerReference{
				{Kind: kind, APIVersion: apiVersion, UID: ownerUID},
			},
		},
	}
}

func podOwnerIndex(kind string) func(client.Object) []string {
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

// countEvents closes a FakeRecorder's channel and returns how many events were emitted.
func countEvents(r *record.FakeRecorder) int {
	close(r.Events)
	n := 0
	for range r.Events {
		n++
	}
	return n
}

func TestPerTypeEvaluation(t *testing.T) {
	const ownUID = types.UID("own-uid")

	type setup struct {
		// ownerKind is the workload kind the controller under test owns.
		ownerKind   string
		ownerKey    string
		ownWorkload client.Object
		// reconcile runs the controller under test against the fake client.
		reconcile func(c client.Client, evaluators []evaluation.Evaluator) (ctrl.Result, *record.FakeRecorder, error)
	}

	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}}

	setups := map[string]setup{
		"DaemonSet": {
			ownerKind: "DaemonSet",
			ownerKey:  daemonSetPodOwnerKey,
			ownWorkload: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: ownUID},
			},
			reconcile: func(c client.Client, ev []evaluation.Evaluator) (ctrl.Result, *record.FakeRecorder, error) {
				rec := record.NewFakeRecorder(10)
				r := &DaemonSetReconciler{Client: c, Evaluators: fakeSource{evs: ev}, Recorder: rec}
				res, err := r.Reconcile(context.Background(), req)
				return res, rec, err
			},
		},
		"ReplicaSet": {
			ownerKind: "ReplicaSet",
			ownerKey:  replicaSetPodOwnerKey,
			ownWorkload: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: ownUID},
			},
			reconcile: func(c client.Client, ev []evaluation.Evaluator) (ctrl.Result, *record.FakeRecorder, error) {
				rec := record.NewFakeRecorder(10)
				r := &ReplicaSetReconciler{Client: c, Evaluators: fakeSource{evs: ev}, Recorder: rec}
				res, err := r.Reconcile(context.Background(), req)
				return res, rec, err
			},
		},
		"StatefulSet": {
			ownerKind: "StatefulSet",
			ownerKey:  statefulSetPodOwnerKey,
			ownWorkload: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: ownUID},
			},
			reconcile: func(c client.Client, ev []evaluation.Evaluator) (ctrl.Result, *record.FakeRecorder, error) {
				rec := record.NewFakeRecorder(10)
				r := &StatefulSetReconciler{Client: c, Evaluators: fakeSource{evs: ev}, Recorder: rec}
				res, err := r.Reconcile(context.Background(), req)
				return res, rec, err
			},
		},
	}

	for name, s := range setups {
		s := s
		t.Run(name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			_ = appsv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)

			pods := []client.Object{
				ownerPod("own", s.ownerKind, "apps/v1", ownUID),
				ownerPod("foreign-group", s.ownerKind, "example.com/v1", ownUID),
			}
			for _, other := range []string{"DaemonSet", "ReplicaSet", "StatefulSet"} {
				if other == s.ownerKind {
					continue
				}
				pods = append(pods, ownerPod("other-"+other, other, "apps/v1", ownUID))
			}

			objs := append([]client.Object{s.ownWorkload}, pods...)
			objs = append(objs, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "default",
					Labels: map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
				},
			})

			c := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objs...).
				WithIndex(&corev1.Pod{}, s.ownerKey, podOwnerIndex(s.ownerKind)).
				Build()

			_, rec, err := s.reconcile(c, []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}})
			if err != nil {
				t.Fatalf("Reconcile() unexpected error: %v", err)
			}

			if got := countEvents(rec); got != 1 {
				t.Errorf("evaluated pod count = %d events, want 1 (only the own apps/v1 pod)", got)
			}
		})
	}
}
