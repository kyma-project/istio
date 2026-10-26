package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
)

const fixedTime = "2026-10-23T10:00:00Z"

func withFixedNow(t *testing.T) {
	t.Helper()
	prev := now
	t.Cleanup(func() { now = prev })
	parsed, err := time.Parse(time.RFC3339, fixedTime)
	if err != nil {
		t.Fatalf("parse fixed time: %v", err)
	}
	now = func() time.Time { return parsed }
}

// injectedNamespace returns the "default" namespace labeled for Istio injection,
// so reconcile passes the ShouldInject gate and reaches evaluation.
func injectedNamespace() *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "default",
			Labels: map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
		},
	}
}

func TestRestartHelpers_StampAnnotation(t *testing.T) {
	withFixedNow(t)

	tests := []struct {
		name    string
		obj     client.Object
		restart func(ctx context.Context, c client.Client, obj client.Object) error
		getTmpl func(client.Object) *corev1.PodTemplateSpec
	}{
		{
			name: "Deployment",
			obj:  &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "d", Namespace: "default"}},
			restart: func(ctx context.Context, c client.Client, o client.Object) error {
				return restartDeployment(ctx, c, o.(*appsv1.Deployment), "")
			},
			getTmpl: func(o client.Object) *corev1.PodTemplateSpec { return &o.(*appsv1.Deployment).Spec.Template },
		},
		{
			name: "StatefulSet",
			obj:  &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"}},
			restart: func(ctx context.Context, c client.Client, o client.Object) error {
				return restartStatefulSet(ctx, c, o.(*appsv1.StatefulSet), "")
			},
			getTmpl: func(o client.Object) *corev1.PodTemplateSpec { return &o.(*appsv1.StatefulSet).Spec.Template },
		},
		{
			name: "DaemonSet",
			obj:  &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "ds", Namespace: "default"}},
			restart: func(ctx context.Context, c client.Client, o client.Object) error {
				return restartDaemonSet(ctx, c, o.(*appsv1.DaemonSet), "")
			},
			getTmpl: func(o client.Object) *corev1.PodTemplateSpec { return &o.(*appsv1.DaemonSet).Spec.Template },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)
			c := fake.NewClientBuilder().WithScheme(s).WithObjects(tt.obj).Build()

			if err := tt.restart(context.Background(), c, tt.obj); err != nil {
				t.Fatalf("restart: %v", err)
			}
			key := client.ObjectKeyFromObject(tt.obj)
			fresh := tt.obj.DeepCopyObject().(client.Object)
			if err := c.Get(context.Background(), key, fresh); err != nil {
				t.Fatalf("get: %v", err)
			}
			got := tt.getTmpl(fresh).Annotations[RestartAnnotation]
			if got != fixedTime {
				t.Errorf("annotation %q = %q, want %q", RestartAnnotation, got, fixedTime)
			}
		})
	}
}

func TestRestartHelpers_PreserveExistingAnnotations(t *testing.T) {
	withFixedNow(t)

	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "d", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"keep": "me"}},
			},
		},
	}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(d).Build()

	if err := restartDeployment(context.Background(), c, d, ""); err != nil {
		t.Fatalf("restart: %v", err)
	}
	fresh := &appsv1.Deployment{}
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(d), fresh); err != nil {
		t.Fatalf("get: %v", err)
	}
	if fresh.Spec.Template.Annotations["keep"] != "me" {
		t.Errorf("existing annotation lost")
	}
	if fresh.Spec.Template.Annotations[RestartAnnotation] != fixedTime {
		t.Errorf("restart annotation missing")
	}
}

func TestRestartReplicaSetPods_DeletesAll(t *testing.T) {
	withFixedNow(t)
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)

	mkPod := func(name string) *corev1.Pod {
		return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}}
	}
	p1 := mkPod("p1")
	p2 := mkPod("p2")
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "default"}}

	c := fake.NewClientBuilder().WithScheme(s).WithObjects(rs, p1, p2).Build()

	if err := restartReplicaSetPods(context.Background(), c, rs, []*corev1.Pod{p1, p2}, "fp-1"); err != nil {
		t.Fatalf("restart: %v", err)
	}

	for _, name := range []string{"p1", "p2"} {
		err := c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: "default"}, &corev1.Pod{})
		if !apierrors.IsNotFound(err) {
			t.Errorf("pod %s expected NotFound, got %v", name, err)
		}
	}

	fresh := &appsv1.ReplicaSet{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "rs", Namespace: "default"}, fresh); err != nil {
		t.Fatalf("get rs: %v", err)
	}
	if fresh.Spec.Template.Annotations[FingerprintAnnotation] != "fp-1" {
		t.Errorf("fingerprint annotation = %q, want %q", fresh.Spec.Template.Annotations[FingerprintAnnotation], "fp-1")
	}
	if fresh.Spec.Template.Annotations[RestartAnnotation] != fixedTime {
		t.Errorf("restart annotation = %q, want %q", fresh.Spec.Template.Annotations[RestartAnnotation], fixedTime)
	}
}

func TestRestartReplicaSetPods_IgnoresNotFound(t *testing.T) {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "default"}}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(rs).Build()

	ghost := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "gone", Namespace: "default"}}
	if err := restartReplicaSetPods(context.Background(), c, rs, []*corev1.Pod{ghost}, "fp-1"); err != nil {
		t.Errorf("expected nil for already-missing pod, got %v", err)
	}
}

// TestReconcile_RestartDecisionTriggersAction proves each workload reconciler
// performs the correct restart on a Restart decision: template annotation for
// Deployment/StatefulSet/DaemonSet, pod delete for ownerless ReplicaSet.
func TestReconcile_RestartDecisionTriggersAction(t *testing.T) {
	withFixedNow(t)

	appLabels := map[string]string{"app": "myapp"}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}}

	tests := []struct {
		name   string
		seed   []client.Object
		index  func(*fake.ClientBuilder) *fake.ClientBuilder
		run    func(c client.Client) error
		assert func(t *testing.T, c client.Client)
	}{
		{
			name: "Deployment template annotation stamped",
			seed: []client.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "dep"},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{MatchLabels: appLabels},
						Replicas: ptr.To(int32(1)),
					},
					Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
				},
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Labels: appLabels}},
			},
			run: func(c client.Client) error {
				r := &DeploymentReconciler{
					Client:     c,
					Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
					Recorder:   record.NewFakeRecorder(10),
				}
				_, err := r.Reconcile(context.Background(), req)
				return err
			},
			assert: func(t *testing.T, c client.Client) {
				got := &appsv1.Deployment{}
				if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
					t.Fatalf("get: %v", err)
				}
				if got.Spec.Template.Annotations[RestartAnnotation] != fixedTime {
					t.Errorf("annotation = %q, want %q", got.Spec.Template.Annotations[RestartAnnotation], fixedTime)
				}
			},
		},
		{
			name: "StatefulSet template annotation stamped",
			seed: []client.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "sts"},
					Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To(int32(1))},
					Status: appsv1.StatefulSetStatus{
						Replicas: 1, ReadyReplicas: 1, CurrentReplicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1,
						CurrentRevision: "r1", UpdateRevision: "r1",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "p", Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{{Kind: "StatefulSet", APIVersion: "apps/v1", UID: "sts"}},
					},
				},
			},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, statefulSetPodOwnerKey, podOwnerIndex("StatefulSet"))
			},
			run: func(c client.Client) error {
				r := &StatefulSetReconciler{
					Client:     c,
					Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
					Recorder:   record.NewFakeRecorder(10),
				}
				_, err := r.Reconcile(context.Background(), req)
				return err
			},
			assert: func(t *testing.T, c client.Client) {
				got := &appsv1.StatefulSet{}
				if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
					t.Fatalf("get: %v", err)
				}
				if got.Spec.Template.Annotations[RestartAnnotation] != fixedTime {
					t.Errorf("missing restart annotation")
				}
			},
		},
		{
			name: "DaemonSet template annotation stamped",
			seed: []client.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "ds"},
					Status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 1, UpdatedNumberScheduled: 1, NumberAvailable: 1,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "p", Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet", APIVersion: "apps/v1", UID: "ds"}},
					},
				},
			},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, daemonSetPodOwnerKey, podOwnerIndex("DaemonSet"))
			},
			run: func(c client.Client) error {
				r := &DaemonSetReconciler{
					Client:     c,
					Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
					Recorder:   record.NewFakeRecorder(10),
				}
				_, err := r.Reconcile(context.Background(), req)
				return err
			},
			assert: func(t *testing.T, c client.Client) {
				got := &appsv1.DaemonSet{}
				if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
					t.Fatalf("get: %v", err)
				}
				if got.Spec.Template.Annotations[RestartAnnotation] != fixedTime {
					t.Errorf("missing restart annotation")
				}
			},
		},
		{
			name: "ownerless ReplicaSet deletes its pods",
			seed: []client.Object{
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "rs"},
					Spec:       appsv1.ReplicaSetSpec{Replicas: ptr.To(int32(1))},
					Status:     appsv1.ReplicaSetStatus{Replicas: 1, AvailableReplicas: 1},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "p", Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "rs"}},
					},
				},
			},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, replicaSetPodOwnerKey, podOwnerIndex("ReplicaSet"))
			},
			run: func(c client.Client) error {
				r := &ReplicaSetReconciler{
					Client:     c,
					Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
					Recorder:   record.NewFakeRecorder(10),
				}
				_, err := r.Reconcile(context.Background(), req)
				return err
			},
			assert: func(t *testing.T, c client.Client) {
				err := c.Get(context.Background(), types.NamespacedName{Name: "p", Namespace: "default"}, &corev1.Pod{})
				if !apierrors.IsNotFound(err) {
					t.Errorf("expected pod deleted (NotFound), got %v", err)
				}
				rs := &appsv1.ReplicaSet{}
				if err := c.Get(context.Background(), req.NamespacedName, rs); err != nil {
					t.Fatalf("get rs: %v", err)
				}
				if _, ok := rs.Spec.Template.Annotations[RestartAnnotation]; !ok {
					t.Errorf("ownerless RS template missing restart annotation")
				}
				if _, ok := rs.Spec.Template.Annotations[FingerprintAnnotation]; !ok {
					t.Errorf("ownerless RS template missing fingerprint annotation")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)
			_ = corev1.AddToScheme(s)
			_ = operatorv1alpha2.AddToScheme(s)
			_ = operatorv1alpha2.AddToScheme(s)

			b := fake.NewClientBuilder().WithScheme(s).WithObjects(append(tt.seed, injectedNamespace())...)
			if tt.index != nil {
				b = tt.index(b)
			}
			c := b.Build()

			if err := tt.run(c); err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			tt.assert(t, c)
		})
	}
}

// TestReconcile_RestartFailurePropagates proves that each reconciler returns
// the restart error so controller-runtime requeues. Uses interceptor funcs to
// force a Patch (template-restart kinds) or Delete (ReplicaSet) error.
func TestReconcile_RestartFailurePropagates(t *testing.T) {
	withFixedNow(t)

	appLabels := map[string]string{"app": "myapp"}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}}
	injected := errors.New("injected")

	patchFail := interceptor.Funcs{
		Patch: func(_ context.Context, _ client.WithWatch, _ client.Object, _ client.Patch, _ ...client.PatchOption) error {
			return injected
		},
	}
	deleteFail := interceptor.Funcs{
		Delete: func(_ context.Context, _ client.WithWatch, _ client.Object, _ ...client.DeleteOption) error {
			return injected
		},
	}

	tests := []struct {
		name  string
		seed  []client.Object
		index func(*fake.ClientBuilder) *fake.ClientBuilder
		funcs interceptor.Funcs
		run   func(c client.Client) error
	}{
		{
			name: "Deployment patch error propagates",
			seed: []client.Object{
				&appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "dep"},
					Spec: appsv1.DeploymentSpec{
						Selector: &metav1.LabelSelector{MatchLabels: appLabels},
						Replicas: ptr.To(int32(1)),
					},
					Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
				},
				&corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Labels: appLabels}},
			},
			funcs: patchFail,
			run: func(c client.Client) error {
				r := &DeploymentReconciler{
					Client:     c,
					Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
					Recorder:   record.NewFakeRecorder(10),
				}
				_, err := r.Reconcile(context.Background(), req)
				return err
			},
		},
		{
			name: "StatefulSet patch error propagates",
			seed: []client.Object{
				&appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "sts"},
					Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To(int32(1))},
					Status: appsv1.StatefulSetStatus{
						Replicas: 1, ReadyReplicas: 1, CurrentReplicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1,
						CurrentRevision: "r1", UpdateRevision: "r1",
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "p", Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{{Kind: "StatefulSet", APIVersion: "apps/v1", UID: "sts"}},
					},
				},
			},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, statefulSetPodOwnerKey, podOwnerIndex("StatefulSet"))
			},
			funcs: patchFail,
			run: func(c client.Client) error {
				r := &StatefulSetReconciler{
					Client:     c,
					Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
					Recorder:   record.NewFakeRecorder(10),
				}
				_, err := r.Reconcile(context.Background(), req)
				return err
			},
		},
		{
			name: "DaemonSet patch error propagates",
			seed: []client.Object{
				&appsv1.DaemonSet{
					ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "ds"},
					Status: appsv1.DaemonSetStatus{
						DesiredNumberScheduled: 1, UpdatedNumberScheduled: 1, NumberAvailable: 1,
					},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "p", Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet", APIVersion: "apps/v1", UID: "ds"}},
					},
				},
			},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, daemonSetPodOwnerKey, podOwnerIndex("DaemonSet"))
			},
			funcs: patchFail,
			run: func(c client.Client) error {
				r := &DaemonSetReconciler{
					Client:     c,
					Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
					Recorder:   record.NewFakeRecorder(10),
				}
				_, err := r.Reconcile(context.Background(), req)
				return err
			},
		},
		{
			name: "ReplicaSet pod delete error propagates",
			seed: []client.Object{
				&appsv1.ReplicaSet{
					ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "rs"},
					Spec:       appsv1.ReplicaSetSpec{Replicas: ptr.To(int32(1))},
					Status:     appsv1.ReplicaSetStatus{Replicas: 1, AvailableReplicas: 1},
				},
				&corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "p", Namespace: "default",
						OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "rs"}},
					},
				},
			},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, replicaSetPodOwnerKey, podOwnerIndex("ReplicaSet"))
			},
			funcs: deleteFail,
			run: func(c client.Client) error {
				r := &ReplicaSetReconciler{
					Client:     c,
					Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
					Recorder:   record.NewFakeRecorder(10),
				}
				_, err := r.Reconcile(context.Background(), req)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)
			_ = corev1.AddToScheme(s)
			_ = operatorv1alpha2.AddToScheme(s)
			_ = operatorv1alpha2.AddToScheme(s)

			b := fake.NewClientBuilder().WithScheme(s).WithObjects(append(tt.seed, injectedNamespace())...).WithInterceptorFuncs(tt.funcs)
			if tt.index != nil {
				b = tt.index(b)
			}
			c := b.Build()

			err := tt.run(c)
			if !errors.Is(err, injected) {
				t.Errorf("expected injected error to propagate, got %v", err)
			}
		})
	}
}

func TestReconcile_ContinueDoesNotRestart(t *testing.T) {
	withFixedNow(t)

	appLabels := map[string]string{"app": "myapp"}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}}

	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "dep"},
		Spec:       appsv1.DeploymentSpec{Selector: &metav1.LabelSelector{MatchLabels: appLabels}},
	}
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Labels: appLabels}}

	c := fake.NewClientBuilder().WithScheme(s).WithObjects(dep, pod, injectedNamespace()).Build()

	r := &DeploymentReconciler{
		Client:     c,
		Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Continue}}},
		Recorder:   record.NewFakeRecorder(10),
	}
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	got := &appsv1.Deployment{}
	if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if _, ok := got.Spec.Template.Annotations[RestartAnnotation]; ok {
		t.Errorf("Continue decision must not stamp restart annotation")
	}
}

// TestReconcile_SkipsRestartWhenAlreadyStamped is the regression guard against
// the restart deadloop: config-drift evaluators keep voting Restart because the
// restart never changes the config they compare, so convergence depends on the
// controller skipping a workload already stamped with the current fingerprint.
// A Restart decision on an already-stamped Deployment must not re-stamp (no new
// rollout); an already-stamped ownerless ReplicaSet must not have its pods
// deleted again.
func TestReconcile_SkipsRestartWhenAlreadyStamped(t *testing.T) {
	withFixedNow(t)

	const fp = "config-fp-abc"
	appLabels := map[string]string{"app": "myapp"}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}}

	t.Run("Deployment already stamped is not restarted again", func(t *testing.T) {
		s := runtime.NewScheme()
		_ = appsv1.AddToScheme(s)
		_ = corev1.AddToScheme(s)
		_ = operatorv1alpha2.AddToScheme(s)

		const priorRestart = "2020-01-01T00:00:00Z"
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "w", Namespace: "default", UID: "dep",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: appLabels},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
						RestartAnnotation:     priorRestart,
						FingerprintAnnotation: fp,
					}},
				},
			},
		}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Labels: appLabels}}
		c := fake.NewClientBuilder().WithScheme(s).WithObjects(dep, pod, injectedNamespace()).Build()

		r := &DeploymentReconciler{
			Client:     c,
			Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}, fp: fp},
			Recorder:   record.NewFakeRecorder(10),
		}
		if _, err := r.Reconcile(context.Background(), req); err != nil {
			t.Fatalf("reconcile: %v", err)
		}

		got := &appsv1.Deployment{}
		if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
			t.Fatalf("get: %v", err)
		}
		// A re-stamp would overwrite the old timestamp with fixedTime.
		if got.Spec.Template.Annotations[RestartAnnotation] != priorRestart {
			t.Errorf("already-stamped workload was restarted again: restartedAt = %q, want unchanged %q",
				got.Spec.Template.Annotations[RestartAnnotation], priorRestart)
		}
	})

	t.Run("ownerless ReplicaSet already stamped keeps its pods", func(t *testing.T) {
		s := runtime.NewScheme()
		_ = appsv1.AddToScheme(s)
		_ = corev1.AddToScheme(s)
		_ = operatorv1alpha2.AddToScheme(s)

		rs := &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: "w", Namespace: "default", UID: "rs",
			},
			Spec: appsv1.ReplicaSetSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
						FingerprintAnnotation: fp,
					}},
				},
			},
		}
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "p", Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: "ReplicaSet", APIVersion: "apps/v1", UID: "rs"}},
			},
		}
		c := fake.NewClientBuilder().WithScheme(s).WithObjects(rs, pod, injectedNamespace()).
			WithIndex(&corev1.Pod{}, replicaSetPodOwnerKey, podOwnerIndex("ReplicaSet")).
			Build()

		r := &ReplicaSetReconciler{
			Client:     c,
			Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}, fp: fp},
			Recorder:   record.NewFakeRecorder(10),
		}
		if _, err := r.Reconcile(context.Background(), req); err != nil {
			t.Fatalf("reconcile: %v", err)
		}

		if err := c.Get(context.Background(), types.NamespacedName{Name: "p", Namespace: "default"}, &corev1.Pod{}); err != nil {
			t.Errorf("already-stamped ReplicaSet deleted its pod again: %v", err)
		}
	})

	t.Run("new fingerprint restarts an already-stamped workload", func(t *testing.T) {
		s := runtime.NewScheme()
		_ = appsv1.AddToScheme(s)
		_ = corev1.AddToScheme(s)
		_ = operatorv1alpha2.AddToScheme(s)

		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name: "w", Namespace: "default", UID: "dep",
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: appLabels},
				Replicas: ptr.To(int32(1)),
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
						RestartAnnotation:     "2020-01-01T00:00:00Z",
						FingerprintAnnotation: "old-fp",
					}},
				},
			},
			Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
		}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Labels: appLabels}}
		c := fake.NewClientBuilder().WithScheme(s).WithObjects(dep, pod, injectedNamespace()).Build()

		r := &DeploymentReconciler{
			Client:     c,
			Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}, fp: fp},
			Recorder:   record.NewFakeRecorder(10),
		}
		if _, err := r.Reconcile(context.Background(), req); err != nil {
			t.Fatalf("reconcile: %v", err)
		}

		got := &appsv1.Deployment{}
		if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
			t.Fatalf("get: %v", err)
		}
		if got.Spec.Template.Annotations[FingerprintAnnotation] != fp {
			t.Errorf("fingerprint not advanced: got %q, want %q", got.Spec.Template.Annotations[FingerprintAnnotation], fp)
		}
		if got.Spec.Template.Annotations[RestartAnnotation] != fixedTime {
			t.Errorf("restart not performed for new fingerprint: restartedAt = %q, want %q",
				got.Spec.Template.Annotations[RestartAnnotation], fixedTime)
		}
	})
}

// restartGateCase describes one workload kind for the skip/defer guard tests:
// a Restart decision that must NOT result in an actual restart because an
// upstream gate (namespace not injected, or rollout still in progress) short-
// circuits reconcile before the restart is performed.
type restartGateCase struct {
	name   string
	seed   []client.Object
	index  func(*fake.ClientBuilder) *fake.ClientBuilder
	run    func(c client.Client) (ctrl.Result, error)
	assert func(t *testing.T, c client.Client, req ctrl.Request)
}

// assertDeploymentNotStamped et al. confirm the restart was skipped.
func assertDeploymentNotStamped(t *testing.T, c client.Client, req ctrl.Request) {
	t.Helper()
	got := &appsv1.Deployment{}
	if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if _, ok := got.Spec.Template.Annotations[RestartAnnotation]; ok {
		t.Errorf("restart annotation present, want workload left unrestarted")
	}
}

func assertStatefulSetNotStamped(t *testing.T, c client.Client, req ctrl.Request) {
	t.Helper()
	got := &appsv1.StatefulSet{}
	if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if _, ok := got.Spec.Template.Annotations[RestartAnnotation]; ok {
		t.Errorf("restart annotation present, want workload left unrestarted")
	}
}

func assertDaemonSetNotStamped(t *testing.T, c client.Client, req ctrl.Request) {
	t.Helper()
	got := &appsv1.DaemonSet{}
	if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
		t.Fatalf("get: %v", err)
	}
	if _, ok := got.Spec.Template.Annotations[RestartAnnotation]; ok {
		t.Errorf("restart annotation present, want workload left unrestarted")
	}
}

func assertReplicaSetPodKept(t *testing.T, c client.Client, _ ctrl.Request) {
	t.Helper()
	if err := c.Get(context.Background(), types.NamespacedName{Name: "p", Namespace: "default"}, &corev1.Pod{}); err != nil {
		t.Errorf("pod was deleted, want it kept: %v", err)
	}
}

// runGateCases builds the fake client for each case (seeding the given
// namespace), runs reconcile asserting a no-error empty result, then runs the
// per-case "no restart happened" assertion.
func runGateCases(t *testing.T, ns *corev1.Namespace, cases []restartGateCase) {
	t.Helper()
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)
			_ = corev1.AddToScheme(s)
			_ = operatorv1alpha2.AddToScheme(s)
			_ = operatorv1alpha2.AddToScheme(s)

			b := fake.NewClientBuilder().WithScheme(s).WithObjects(append(tt.seed, ns)...)
			if tt.index != nil {
				b = tt.index(b)
			}
			c := b.Build()

			res, err := tt.run(c)
			if err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			if res != (ctrl.Result{}) {
				t.Errorf("result = %v, want empty", res)
			}
			tt.assert(t, c, req)
		})
	}
}

// restartGateSeeds returns one rolled-out workload (+ owned/matching pod) per
// kind so that, with an injected namespace, a Restart decision would fire; the
// skip/defer tests vary the namespace or the rollout status to prove the gate.
func restartGateSeeds(rolledOut bool) []restartGateCase {
	appLabels := map[string]string{"app": "myapp"}

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "dep"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: appLabels},
			Replicas: ptr.To(int32(1)),
		},
	}
	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "sts"},
		Spec:       appsv1.StatefulSetSpec{Replicas: ptr.To(int32(1))},
	}
	ds := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "ds"},
	}
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "rs"},
		Spec:       appsv1.ReplicaSetSpec{Replicas: ptr.To(int32(1))},
	}

	if rolledOut {
		dep.Status = appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1}
		sts.Status = appsv1.StatefulSetStatus{
			Replicas: 1, ReadyReplicas: 1, CurrentReplicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1,
			CurrentRevision: "r1", UpdateRevision: "r1",
		}
		ds.Status = appsv1.DaemonSetStatus{DesiredNumberScheduled: 1, UpdatedNumberScheduled: 1, NumberAvailable: 1}
		rs.Status = appsv1.ReplicaSetStatus{Replicas: 1, AvailableReplicas: 1}
	} else {
		// A rollout is in progress: Generation has outrun ObservedGeneration, so
		// every rolledOut predicate returns false regardless of replica counts.
		dep.Generation, dep.Status = 2, appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1}
		sts.Generation, sts.Status = 2, appsv1.StatefulSetStatus{
			ObservedGeneration: 1, Replicas: 1, ReadyReplicas: 1, CurrentReplicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1,
			CurrentRevision: "r1", UpdateRevision: "r1",
		}
		ds.Generation, ds.Status = 2, appsv1.DaemonSetStatus{ObservedGeneration: 1, DesiredNumberScheduled: 1, UpdatedNumberScheduled: 1, NumberAvailable: 1}
		rs.Generation, rs.Status = 2, appsv1.ReplicaSetStatus{ObservedGeneration: 1, Replicas: 1, AvailableReplicas: 1}
	}

	ownedPod := func(kind string, uid types.UID) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name: "p", Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{{Kind: kind, APIVersion: "apps/v1", UID: uid}},
			},
		}
	}
	restartEvaluators := fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}}

	return []restartGateCase{
		{
			name: "Deployment",
			seed: []client.Object{dep, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Labels: appLabels}}},
			run: func(c client.Client) (ctrl.Result, error) {
				r := &DeploymentReconciler{Client: c, Evaluators: restartEvaluators, Recorder: record.NewFakeRecorder(10)}
				return r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}})
			},
			assert: assertDeploymentNotStamped,
		},
		{
			name: "StatefulSet",
			seed: []client.Object{sts, ownedPod("StatefulSet", "sts")},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, statefulSetPodOwnerKey, podOwnerIndex("StatefulSet"))
			},
			run: func(c client.Client) (ctrl.Result, error) {
				r := &StatefulSetReconciler{Client: c, Evaluators: restartEvaluators, Recorder: record.NewFakeRecorder(10)}
				return r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}})
			},
			assert: assertStatefulSetNotStamped,
		},
		{
			name: "DaemonSet",
			seed: []client.Object{ds, ownedPod("DaemonSet", "ds")},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, daemonSetPodOwnerKey, podOwnerIndex("DaemonSet"))
			},
			run: func(c client.Client) (ctrl.Result, error) {
				r := &DaemonSetReconciler{Client: c, Evaluators: restartEvaluators, Recorder: record.NewFakeRecorder(10)}
				return r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}})
			},
			assert: assertDaemonSetNotStamped,
		},
		{
			name: "ReplicaSet",
			seed: []client.Object{rs, ownedPod("ReplicaSet", "rs")},
			index: func(b *fake.ClientBuilder) *fake.ClientBuilder {
				return b.WithIndex(&corev1.Pod{}, replicaSetPodOwnerKey, podOwnerIndex("ReplicaSet"))
			},
			run: func(c client.Client) (ctrl.Result, error) {
				r := &ReplicaSetReconciler{Client: c, Evaluators: restartEvaluators, Recorder: record.NewFakeRecorder(10)}
				return r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}})
			},
			assert: assertReplicaSetPodKept,
		},
	}
}

// TestReconcile_NonInjectedNamespaceIsSkipped proves that a workload in a
// namespace not selected for Istio injection is skipped before evaluation, so a
// Restart decision never results in a restart even though the workload is rolled
// out.
func TestReconcile_NonInjectedNamespaceIsSkipped(t *testing.T) {
	withFixedNow(t)
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "default",
			Labels: map[string]string{evaluation.InjectionLabel: evaluation.InjectDisabled},
		},
	}
	runGateCases(t, ns, restartGateSeeds(true))
}

// TestReconcile_RolloutInProgressDefersRestart proves that a Restart decision on
// an injected but not-yet-rolled-out workload defers the restart: no stamp, no
// pod delete, and an empty result (the next reconcile retries once rolled out).
func TestReconcile_RolloutInProgressDefersRestart(t *testing.T) {
	withFixedNow(t)
	runGateCases(t, injectedNamespace(), restartGateSeeds(false))
}
