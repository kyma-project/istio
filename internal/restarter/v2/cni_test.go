package v2

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestCNIConfigChanged_Evaluate(t *testing.T) {
	tc := []struct {
		name           string
		disabled       bool
		object         client.Object
		expectDecision Decision
	}{
		{
			name:     "cni enabled and istio-init present, restart",
			disabled: false,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "istio-init"}},
				},
			},
			expectDecision: Restart,
		},
		{
			name:     "cni enabled and istio-validation present, continue",
			disabled: false,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "istio-validation"}},
				},
			},
			expectDecision: Continue,
		},
		{
			name:     "cni disabled and istio-validation present, restart",
			disabled: true,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "istio-validation"}},
				},
			},
			expectDecision: Restart,
		},
		{
			name:     "cni disabled and istio-init present, continue",
			disabled: true,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "istio-init"}},
				},
			},
			expectDecision: Continue,
		},
		{
			name:     "no matching init container, continue",
			disabled: false,
			object: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"},
				Spec: corev1.PodSpec{
					InitContainers: []corev1.Container{{Name: "app-init"}},
				},
			},
			expectDecision: Continue,
		},
		{
			name:           "object not a pod, continue",
			disabled:       false,
			object:         &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"}},
			expectDecision: Continue,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewCNIChangedRule(tt.disabled)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}
