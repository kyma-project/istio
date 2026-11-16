package evaluation_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestPrometheusMergeConfigChanged_Evaluate(t *testing.T) {
	const port int32 = 15020

	tc := []struct {
		name           string
		enabled        bool
		object         client.Object
		expectDecision evaluation.Decision
	}{
		{
			name:    "enabled and annotations match, continue",
			enabled: true,
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"prometheus.io/path": "/stats/prometheus",
				"prometheus.io/port": "15020",
			}}},
			expectDecision: evaluation.Continue,
		},
		{
			name:           "enabled and annotations missing, restart",
			enabled:        true,
			object:         &corev1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			expectDecision: evaluation.Restart,
		},
		{
			name:    "enabled and port differs, restart",
			enabled: true,
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"prometheus.io/path": "/stats/prometheus",
				"prometheus.io/port": "9090",
			}}},
			expectDecision: evaluation.Restart,
		},
		{
			name:           "disabled and annotations absent, continue",
			enabled:        false,
			object:         &corev1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			expectDecision: evaluation.Continue,
		},
		{
			name:    "disabled and annotations present, restart",
			enabled: false,
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"prometheus.io/path": "/stats/prometheus",
				"prometheus.io/port": "15020",
			}}},
			expectDecision: evaluation.Restart,
		},
		{
			name:           "object not a pod, continue",
			enabled:        true,
			object:         &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"}},
			expectDecision: evaluation.Continue,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			eval := evaluation.NewPrometheusMergeConfigChanged(tt.enabled, port)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}
