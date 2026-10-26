package controller

import (
	"testing"

	"github.com/kyma-project/istio/operator/internal/restarter"
)

func TestSelectRestarters(t *testing.T) {
	all := []restarter.Restarter{nil, nil, nil}

	tests := []struct {
		name          string
		enableRestart bool
		input         []restarter.Restarter
		wantLen       int
	}{
		{
			name:          "returns all restarters when enabled",
			enableRestart: true,
			input:         all,
			wantLen:       3,
		},
		{
			name:          "returns no restarters when disabled",
			enableRestart: false,
			input:         all,
			wantLen:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := selectRestarters(tt.enableRestart, tt.input)
			if len(got) != tt.wantLen {
				t.Errorf("selectRestarters(%v) returned %d restarters, want %d", tt.enableRestart, len(got), tt.wantLen)
			}
		})
	}
}
