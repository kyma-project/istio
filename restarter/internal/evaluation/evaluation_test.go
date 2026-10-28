package evaluation_test

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/restarter/internal/evaluation"
)

// stubEvaluator returns a fixed Decision for every call.
type stubEvaluator struct{ d evaluation.Decision }

func (s *stubEvaluator) Name() string                                 { return "stubEvaluator" }
func (s *stubEvaluator) Evaluate(_ client.Object) evaluation.Decision { return s.d }

func pod(name string) *corev1.Pod {
	return &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns"}}
}

func deployment(name string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns"}}
}

// typedStubEvaluator returns podDecision for *corev1.Pod objects, workloadDecision otherwise.
type typedStubEvaluator struct {
	workloadDecision evaluation.Decision
	podDecision      evaluation.Decision
}

func (s *typedStubEvaluator) Name() string { return "typedStubEvaluator" }
func (s *typedStubEvaluator) Evaluate(obj client.Object) evaluation.Decision {
	if _, ok := obj.(*corev1.Pod); ok {
		return s.podDecision
	}
	return s.workloadDecision
}

func TestEvaluateDetailed(t *testing.T) {
	tests := []struct {
		name       string
		workload   client.Object
		pods       []*corev1.Pod
		evaluators []evaluation.Evaluator
		wantD      evaluation.Decision
		wantK      evaluation.RestartKind
	}{
		{
			name:       "no pods returns Continue KindNone",
			workload:   deployment("dep"),
			pods:       nil,
			evaluators: []evaluation.Evaluator{&stubEvaluator{evaluation.Restart}},
			wantD:      evaluation.Continue,
			wantK:      evaluation.KindNone,
		},
		{
			name:     "workload-level Restart returns KindConfigDrift",
			workload: deployment("dep"),
			pods:     []*corev1.Pod{pod("p1")},
			evaluators: []evaluation.Evaluator{
				&typedStubEvaluator{workloadDecision: evaluation.Restart, podDecision: evaluation.Continue},
			},
			wantD: evaluation.Restart,
			wantK: evaluation.KindConfigDrift,
		},
		{
			name:     "pod-level Restart returns KindPodState",
			workload: deployment("dep"),
			pods:     []*corev1.Pod{pod("p1")},
			evaluators: []evaluation.Evaluator{
				&typedStubEvaluator{workloadDecision: evaluation.Continue, podDecision: evaluation.Restart},
			},
			wantD: evaluation.Restart,
			wantK: evaluation.KindPodState,
		},
		{
			name:     "Stop returns KindNone",
			workload: deployment("dep"),
			pods:     []*corev1.Pod{pod("p1")},
			evaluators: []evaluation.Evaluator{
				&stubEvaluator{evaluation.Stop},
			},
			wantD: evaluation.Stop,
			wantK: evaluation.KindNone,
		},
		{
			name:     "all Continue returns KindNone",
			workload: deployment("dep"),
			pods:     []*corev1.Pod{pod("p1")},
			evaluators: []evaluation.Evaluator{
				&stubEvaluator{evaluation.Continue},
			},
			wantD: evaluation.Continue,
			wantK: evaluation.KindNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotD, gotK := evaluation.EvaluateDetailed(context.Background(), tt.workload, tt.pods, tt.evaluators)
			if gotD != tt.wantD {
				t.Errorf("EvaluateDetailed() decision = %v, want %v", gotD, tt.wantD)
			}
			if gotK != tt.wantK {
				t.Errorf("EvaluateDetailed() kind = %v, want %v", gotK, tt.wantK)
			}
		})
	}
}
