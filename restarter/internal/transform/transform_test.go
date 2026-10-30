package transform

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStripForRestart(t *testing.T) {
	strip := StripForRestart()

	tests := []struct {
		name string
		in   any
	}{
		{
			name: "deployment",
			in: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "dep",
					Namespace:   "ns",
					Labels:      map[string]string{"app": "x"},
					Annotations: map[string]string{"a": "b"},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{Manager: "m"},
					},
					OwnerReferences: []metav1.OwnerReference{
						{Name: "owner", APIVersion: "v1", Kind: "Deployment", UID: "uid"},
					},
					Finalizers: []string{"protect.me"},
				},
			},
		},
		{
			name: "pod",
			in: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Namespace:   "ns",
					Labels:      map[string]string{"app": "x"},
					Annotations: map[string]string{"a": "b"},
					ManagedFields: []metav1.ManagedFieldsEntry{
						{Manager: "m"},
					},
					OwnerReferences: []metav1.OwnerReference{
						{Name: "owner", APIVersion: "apps/v1", Kind: "ReplicaSet", UID: "uid"},
					},
					Finalizers: []string{"protect.me"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := strip(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			acc, err := meta.Accessor(out)
			if err != nil {
				t.Fatalf("output not a metav1.Object: %v", err)
			}

			if len(acc.GetManagedFields()) != 0 {
				t.Errorf("managedFields not stripped: %v", acc.GetManagedFields())
			}
			if len(acc.GetOwnerReferences()) != 0 {
				t.Errorf("ownerReferences not stripped: %v", acc.GetOwnerReferences())
			}
			if len(acc.GetFinalizers()) == 0 {
				t.Errorf("finalizers must be retained, got none")
			}
			if acc.GetName() == "" || acc.GetNamespace() == "" {
				t.Errorf("name/namespace must be retained, got name=%q ns=%q", acc.GetName(), acc.GetNamespace())
			}
			if acc.GetLabels()["app"] != "x" {
				t.Errorf("labels not retained: %v", acc.GetLabels())
			}
			if acc.GetAnnotations()["a"] != "b" {
				t.Errorf("annotations not retained: %v", acc.GetAnnotations())
			}
		})
	}
}

func TestStripForRestart_NonObjectPassthrough(t *testing.T) {
	strip := StripForRestart()
	in := "not-a-k8s-object"

	out, err := strip(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != in {
		t.Errorf("non-object should pass through unchanged, got %v", out)
	}
}
