package predicates_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podWithInitContainers(names ...string) v1.Pod {
	containers := make([]v1.Container, 0, len(names))
	for _, name := range names {
		containers = append(containers, v1.Container{Name: name})
	}
	return v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "test-pod"},
		Spec:       v1.PodSpec{InitContainers: containers},
	}
}

func TestCniRestartPredicate_Matches(t *testing.T) {
	// When CNI is enabled (disableCni: false), pods must use istio-validation.
	// Pods that still have istio-init are stale and must be restarted.
	//
	// When CNI is disabled (disableCni: true), pods must use istio-init.
	// Pods that still have istio-validation are stale and must be restarted.
	tests := []struct {
		name          string
		disableCni    bool
		pod           v1.Pod
		expectRestart bool
	}{
		// --- CNI enabled (disableCni: false) ---
		{
			name:          "cni enabled: pod with istio-init should restart",
			disableCni:    false,
			pod:           podWithInitContainers("istio-init"),
			expectRestart: true,
		},
		{
			name:          "cni enabled: pod with istio-validation should not restart",
			disableCni:    false,
			pod:           podWithInitContainers("istio-validation"),
			expectRestart: false,
		},
		{
			name:          "cni enabled: pod with unrelated init container should not restart",
			disableCni:    false,
			pod:           podWithInitContainers("some-other-init"),
			expectRestart: false,
		},
		{
			name:          "cni enabled: pod with no init containers should not restart",
			disableCni:    false,
			pod:           v1.Pod{},
			expectRestart: false,
		},
		{
			name:          "cni enabled: pod with istio-init among multiple init containers should restart",
			disableCni:    false,
			pod:           podWithInitContainers("some-other-init", "istio-init"),
			expectRestart: true,
		},
		// --- CNI disabled (disableCni: true) ---
		{
			name:          "cni disabled: pod with istio-validation should restart",
			disableCni:    true,
			pod:           podWithInitContainers("istio-validation"),
			expectRestart: true,
		},
		{
			name:          "cni disabled: pod with istio-init should not restart",
			disableCni:    true,
			pod:           podWithInitContainers("istio-init"),
			expectRestart: false,
		},
		{
			name:          "cni disabled: pod with unrelated init container should not restart",
			disableCni:    true,
			pod:           podWithInitContainers("some-other-init"),
			expectRestart: false,
		},
		{
			name:          "cni disabled: pod with no init containers should not restart",
			disableCni:    true,
			pod:           v1.Pod{},
			expectRestart: false,
		},
		{
			name:          "cni disabled: pod with istio-validation among multiple init containers should restart",
			disableCni:    true,
			pod:           podWithInitContainers("some-other-init", "istio-validation"),
			expectRestart: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predicate := predicates.NewCniRestartPredicate(tt.disableCni)
			assert.Equal(t, tt.expectRestart, predicate.Matches(tt.pod))
		})
	}
}

func TestCniRestartPredicate_MustMatch(t *testing.T) {
	// MustMatch must return false — the CNI predicate is a filter, not a requirement.
	// Pods without any istio init container are simply not selected, not an error.
	predicate := predicates.NewCniRestartPredicate(false)
	assert.False(t, predicate.MustMatch())
}

func TestCniRestartPredicate_Name(t *testing.T) {
	predicate := predicates.NewCniRestartPredicate(false)
	assert.Equal(t, "CniRestartPredicate", predicate.Name())
}
