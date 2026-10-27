package restart

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func assertRestartAnnotationRecent(t *testing.T, got string) {
	t.Helper()
	if got == "" {
		t.Fatalf("restart annotation missing")
	}
	parsed, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("restart annotation %q not RFC3339: %v", got, err)
	}
	if d := time.Since(parsed); d < -5*time.Second || d > 30*time.Second {
		t.Errorf("restart annotation %q not recent (delta %v)", got, d)
	}
}

func TestRestartHelpers_StampAnnotation(t *testing.T) {
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
				return Deployment(ctx, c, o.(*appsv1.Deployment), "")
			},
			getTmpl: func(o client.Object) *corev1.PodTemplateSpec { return &o.(*appsv1.Deployment).Spec.Template },
		},
		{
			name: "StatefulSet",
			obj:  &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"}},
			restart: func(ctx context.Context, c client.Client, o client.Object) error {
				return StatefulSet(ctx, c, o.(*appsv1.StatefulSet), "")
			},
			getTmpl: func(o client.Object) *corev1.PodTemplateSpec { return &o.(*appsv1.StatefulSet).Spec.Template },
		},
		{
			name: "DaemonSet",
			obj:  &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "ds", Namespace: "default"}},
			restart: func(ctx context.Context, c client.Client, o client.Object) error {
				return DaemonSet(ctx, c, o.(*appsv1.DaemonSet), "")
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
			assertRestartAnnotationRecent(t, tt.getTmpl(fresh).Annotations[RestartAnnotation])
		})
	}
}

func TestRestartHelpers_PreserveExistingAnnotations(t *testing.T) {
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

	if err := Deployment(context.Background(), c, d, ""); err != nil {
		t.Fatalf("restart: %v", err)
	}
	fresh := &appsv1.Deployment{}
	if err := c.Get(context.Background(), client.ObjectKeyFromObject(d), fresh); err != nil {
		t.Fatalf("get: %v", err)
	}
	if fresh.Spec.Template.Annotations["keep"] != "me" {
		t.Errorf("existing annotation lost")
	}
	assertRestartAnnotationRecent(t, fresh.Spec.Template.Annotations[RestartAnnotation])
}

func TestRestartReplicaSetPods_DeletesAll(t *testing.T) {
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

	if err := ReplicaSetPods(context.Background(), c, rs, []*corev1.Pod{p1, p2}, "fp-1"); err != nil {
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
	if fresh.Spec.Template.Annotations[HashAnnotation] != "fp-1" {
		t.Errorf("fingerprint annotation = %q, want %q", fresh.Spec.Template.Annotations[HashAnnotation], "fp-1")
	}
	assertRestartAnnotationRecent(t, fresh.Spec.Template.Annotations[RestartAnnotation])
}

func TestRestartReplicaSetPods_IgnoresNotFound(t *testing.T) {
	s := runtime.NewScheme()
	_ = corev1.AddToScheme(s)
	_ = appsv1.AddToScheme(s)
	rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "default"}}
	c := fake.NewClientBuilder().WithScheme(s).WithObjects(rs).Build()

	ghost := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "gone", Namespace: "default"}}
	if err := ReplicaSetPods(context.Background(), c, rs, []*corev1.Pod{ghost}, "fp-1"); err != nil {
		t.Errorf("expected nil for already-missing pod, got %v", err)
	}
}
