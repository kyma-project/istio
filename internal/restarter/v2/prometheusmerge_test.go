package v2

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPrometheusMergeConfigChanged_Evaluate(t *testing.T) {
	const port int32 = 15020

	tc := []struct {
		name           string
		enabled        bool
		object         client.Object
		expectDecision Decision
	}{
		{
			name:    "enabled and annotations match, continue",
			enabled: true,
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"prometheus.io/path": "/stats/prometheus",
				"prometheus.io/port": "15020",
			}}},
			expectDecision: Continue,
		},
		{
			name:           "enabled and annotations missing, restart",
			enabled:        true,
			object:         &corev1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			expectDecision: Restart,
		},
		{
			name:    "enabled and port differs, restart",
			enabled: true,
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"prometheus.io/path": "/stats/prometheus",
				"prometheus.io/port": "9090",
			}}},
			expectDecision: Restart,
		},
		{
			name:           "disabled and annotations absent, continue",
			enabled:        false,
			object:         &corev1.Pod{ObjectMeta: metav1.ObjectMeta{}},
			expectDecision: Continue,
		},
		{
			name:    "disabled and annotations present, restart",
			enabled: false,
			object: &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
				"prometheus.io/path": "/stats/prometheus",
				"prometheus.io/port": "15020",
			}}},
			expectDecision: Restart,
		},
		{
			name:           "object not a pod, continue",
			enabled:        true,
			object:         &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "foo", Namespace: "bar"}},
			expectDecision: Continue,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			eval := NewPrometheusMergeConfigChanged(tt.enabled, port)
			d := eval.Evaluate(tt.object)
			if d != tt.expectDecision {
				t.Errorf("wrong decision: got %v, want %v", d, tt.expectDecision)
			}
		})
	}
}

func TestGetStatusPort(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add corev1 to scheme: %v", err)
	}

	configMap := func(data map[string]string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "istio-system", Name: "istio"},
			Data:       data,
		}
	}

	tc := []struct {
		name       string
		objects    []client.Object
		expectPort int32
	}{
		{
			name:       "configmap missing, default port",
			objects:    nil,
			expectPort: 15020,
		},
		{
			name:       "configmap without mesh key, default port",
			objects:    []client.Object{configMap(map[string]string{"other": "value"})},
			expectPort: 15020,
		},
		{
			name:       "invalid mesh config, default port",
			objects:    []client.Object{configMap(map[string]string{"mesh": "\t not: [valid"})},
			expectPort: 15020,
		},
		{
			name:       "mesh config without status port, default port",
			objects:    []client.Object{configMap(map[string]string{"mesh": "defaultConfig: {}"})},
			expectPort: 15020,
		},
		{
			name:       "mesh config with custom status port",
			objects:    []client.Object{configMap(map[string]string{"mesh": "defaultConfig:\n  statusPort: 25020"})},
			expectPort: 25020,
		},
	}
	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tt.objects...).Build()
			got := GetStatusPort(context.Background(), c)
			if got != tt.expectPort {
				t.Errorf("wrong port: got %d, want %d", got, tt.expectPort)
			}
		})
	}
}
