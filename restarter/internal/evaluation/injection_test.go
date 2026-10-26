package evaluation_test

import (
	"testing"

	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
)

func TestShouldInject(t *testing.T) {
	tests := []struct {
		name        string
		nsLabels    map[string]string
		labels      map[string]string
		annotations map[string]string
		want        bool
	}{
		{
			name:     "enabled namespace injects",
			nsLabels: map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
			want:     true,
		},
		{
			name:     "disabled namespace does not inject",
			nsLabels: map[string]string{evaluation.InjectionLabel: evaluation.InjectDisabled},
			want:     false,
		},
		{
			name:     "unlabeled namespace does not inject",
			nsLabels: nil,
			want:     false,
		},
		{
			name:        "annotation true injects in unlabeled namespace",
			nsLabels:    nil,
			annotations: map[string]string{evaluation.InjectOverride: "true"},
			want:        true,
		},
		{
			name:        "annotation true injects in enabled namespace",
			nsLabels:    map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
			annotations: map[string]string{evaluation.InjectOverride: "true"},
			want:        true,
		},
		{
			name:        "annotation true does not override disabled namespace",
			nsLabels:    map[string]string{evaluation.InjectionLabel: evaluation.InjectDisabled},
			annotations: map[string]string{evaluation.InjectOverride: "true"},
			want:        false,
		},
		{
			name:        "annotation false opts out of enabled namespace",
			nsLabels:    map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
			annotations: map[string]string{evaluation.InjectOverride: "false"},
			want:        false,
		},
		{
			name:     "label true injects in unlabeled namespace",
			nsLabels: nil,
			labels:   map[string]string{evaluation.InjectOverride: "true"},
			want:     true,
		},
		{
			name:     "label false opts out of enabled namespace",
			nsLabels: map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
			labels:   map[string]string{evaluation.InjectOverride: "false"},
			want:     false,
		},
		{
			name:        "conflicting label true and annotation false opts out",
			nsLabels:    map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
			labels:      map[string]string{evaluation.InjectOverride: "true"},
			annotations: map[string]string{evaluation.InjectOverride: "false"},
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluation.ShouldInject(tt.nsLabels, tt.labels, tt.annotations); got != tt.want {
				t.Errorf("ShouldInject() = %v, want %v", got, tt.want)
			}
		})
	}
}
