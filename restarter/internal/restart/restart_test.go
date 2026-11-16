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

func TestDeployment(t *testing.T) {
	tests := []struct {
		name        string
		existing    map[string]string
		fingerprint string
	}{
		{name: "stamps restart and hash", fingerprint: "fp-dep"},
		{name: "preserves existing annotations", existing: map[string]string{"keep": "me"}, fingerprint: "fp-dep"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)

			d := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "d", Namespace: "default"},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{Annotations: tt.existing},
					},
				},
			}
			c := fake.NewClientBuilder().WithScheme(s).WithObjects(d).Build()

			if err := Deployment(context.Background(), c, d, tt.fingerprint); err != nil {
				t.Fatalf("Deployment: %v", err)
			}

			fresh := &appsv1.Deployment{}
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(d), fresh); err != nil {
				t.Fatalf("get: %v", err)
			}
			ann := fresh.Spec.Template.Annotations
			if ann[HashAnnotation] != tt.fingerprint {
				t.Errorf("hash annotation = %q, want %q", ann[HashAnnotation], tt.fingerprint)
			}
			for k, v := range tt.existing {
				if ann[k] != v {
					t.Errorf("existing annotation %q = %q, want %q", k, ann[k], v)
				}
			}
			assertRestartAnnotationRecent(t, ann[RestartAnnotation])
		})
	}
}

func TestStatefulSet(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint string
	}{
		{name: "stamps restart and hash", fingerprint: "fp-sts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)

			sts := &appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "s", Namespace: "default"}}
			c := fake.NewClientBuilder().WithScheme(s).WithObjects(sts).Build()

			if err := StatefulSet(context.Background(), c, sts, tt.fingerprint); err != nil {
				t.Fatalf("StatefulSet: %v", err)
			}

			fresh := &appsv1.StatefulSet{}
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(sts), fresh); err != nil {
				t.Fatalf("get: %v", err)
			}
			if fresh.Spec.Template.Annotations[HashAnnotation] != tt.fingerprint {
				t.Errorf("hash annotation = %q, want %q", fresh.Spec.Template.Annotations[HashAnnotation], tt.fingerprint)
			}
			assertRestartAnnotationRecent(t, fresh.Spec.Template.Annotations[RestartAnnotation])
		})
	}
}

func TestDaemonSet(t *testing.T) {
	tests := []struct {
		name        string
		fingerprint string
	}{
		{name: "stamps restart and hash", fingerprint: "fp-ds"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)

			ds := &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "ds", Namespace: "default"}}
			c := fake.NewClientBuilder().WithScheme(s).WithObjects(ds).Build()

			if err := DaemonSet(context.Background(), c, ds, tt.fingerprint); err != nil {
				t.Fatalf("DaemonSet: %v", err)
			}

			fresh := &appsv1.DaemonSet{}
			if err := c.Get(context.Background(), client.ObjectKeyFromObject(ds), fresh); err != nil {
				t.Fatalf("get: %v", err)
			}
			if fresh.Spec.Template.Annotations[HashAnnotation] != tt.fingerprint {
				t.Errorf("hash annotation = %q, want %q", fresh.Spec.Template.Annotations[HashAnnotation], tt.fingerprint)
			}
			assertRestartAnnotationRecent(t, fresh.Spec.Template.Annotations[RestartAnnotation])
		})
	}
}

func TestReplicaSetPods(t *testing.T) {
	tests := []struct {
		name        string
		podNames    []string
		targetNames []string
		wantDeleted []string
		checkStamp  bool
	}{
		{
			name:        "deletes all targeted pods and stamps replicaset",
			podNames:    []string{"p1", "p2"},
			targetNames: []string{"p1", "p2"},
			wantDeleted: []string{"p1", "p2"},
			checkStamp:  true,
		},
		{
			name:        "ignores already-missing pod",
			podNames:    nil,
			targetNames: []string{"gone"},
			wantDeleted: nil,
			checkStamp:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = corev1.AddToScheme(s)
			_ = appsv1.AddToScheme(s)

			rs := &appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "default"}}
			objs := []client.Object{rs}
			byName := map[string]*corev1.Pod{}
			for _, name := range tt.podNames {
				p := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}}
				byName[name] = p
				objs = append(objs, p)
			}

			c := fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()

			targets := make([]*corev1.Pod, 0, len(tt.targetNames))
			for _, name := range tt.targetNames {
				if p, ok := byName[name]; ok {
					targets = append(targets, p)
					continue
				}
				targets = append(targets, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default"}})
			}

			if err := ReplicaSetPods(context.Background(), c, rs, targets, "fp-1"); err != nil {
				t.Fatalf("ReplicaSetPods: %v", err)
			}

			for _, name := range tt.wantDeleted {
				err := c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: "default"}, &corev1.Pod{})
				if !apierrors.IsNotFound(err) {
					t.Errorf("pod %s expected NotFound, got %v", name, err)
				}
			}

			if tt.checkStamp {
				fresh := &appsv1.ReplicaSet{}
				if err := c.Get(context.Background(), types.NamespacedName{Name: "rs", Namespace: "default"}, fresh); err != nil {
					t.Fatalf("get rs: %v", err)
				}
				if fresh.Spec.Template.Annotations[HashAnnotation] != "fp-1" {
					t.Errorf("fingerprint annotation = %q, want %q", fresh.Spec.Template.Annotations[HashAnnotation], "fp-1")
				}
				assertRestartAnnotationRecent(t, fresh.Spec.Template.Annotations[RestartAnnotation])
			}
		})
	}
}

func TestAlreadyRestartedForConfig(t *testing.T) {
	tests := []struct {
		name    string
		stamped string
		hash    string
		want    bool
	}{
		{name: "empty hash with stamped", stamped: "abc", hash: "", want: false},
		{name: "empty hash and empty stamped", stamped: "", hash: "", want: false},
		{name: "stamped equals hash", stamped: "abc", hash: "abc", want: true},
		{name: "stamped differs from hash", stamped: "abc", hash: "def", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := AlreadyRestartedForConfig(tt.stamped, tt.hash); got != tt.want {
				t.Errorf("AlreadyRestartedForConfig(%q, %q) = %v, want %v", tt.stamped, tt.hash, got, tt.want)
			}
		})
	}
}
