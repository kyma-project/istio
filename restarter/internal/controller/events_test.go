package controller

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/record"

	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
)

func newFakeDeployment() *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deploy", Namespace: "default"},
	}
}

func TestEmitDecisionEvent(t *testing.T) {
	tests := []struct {
		name     string
		decision evaluation.Decision
		wantEvt  string // empty = no event expected
	}{
		{
			name:     "Restart emits warning",
			decision: evaluation.Restart,
			wantEvt:  "Warning RestartRequired workload pods require a restart",
		},
		{
			name:     "Stop emits normal",
			decision: evaluation.Stop,
			wantEvt:  "Normal RestartSkipped restart skipped: evaluation returned Stop",
		},
		{
			name:     "Continue emits nothing",
			decision: evaluation.Continue,
			wantEvt:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := record.NewFakeRecorder(10)
			emitDecisionEvent(rec, newFakeDeployment(), tt.decision)

			select {
			case got := <-rec.Events:
				if tt.wantEvt == "" {
					t.Errorf("expected no event, got %q", got)
				} else if got != tt.wantEvt {
					t.Errorf("got %q, want %q", got, tt.wantEvt)
				}
			default:
				if tt.wantEvt != "" {
					t.Fatalf("expected event %q but channel was empty", tt.wantEvt)
				}
			}
		})
	}
}
