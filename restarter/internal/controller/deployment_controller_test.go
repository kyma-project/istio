package controller

import (
	"context"
	"errors"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/restarter/internal/evaluation"
)

func makeDeploymentWithSelector(uid types.UID, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-dep",
			Namespace: "default",
			UID:       uid,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
		},
	}
}

func makePodWithLabels(name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels:    labels,
		},
	}
}

func TestDeploymentReconcile(t *testing.T) {
	appLabels := map[string]string{"app": "myapp"}

	tests := []struct {
		name           string
		deployment     *appsv1.Deployment
		pods           []*corev1.Pod
		evaluators     []evaluation.Evaluator
		snapshotErr    error
		wantResult     ctrl.Result
		wantErrPresent bool
	}{
		{
			name:       "no pods, no evaluators → Continue, no error",
			deployment: makeDeploymentWithSelector("dep-uid-1", appLabels),
			pods:       nil,
			evaluators: nil,
			wantResult: ctrl.Result{},
		},
		{
			name:       "matching pods, evaluator returns Continue",
			deployment: makeDeploymentWithSelector("dep-uid-1", appLabels),
			pods:       []*corev1.Pod{makePodWithLabels("p1", appLabels)},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Continue}},
			wantResult: ctrl.Result{},
		},
		{
			name:       "matching pods, evaluator returns Restart",
			deployment: makeDeploymentWithSelector("dep-uid-1", appLabels),
			pods:       []*corev1.Pod{makePodWithLabels("p1", appLabels)},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			wantResult: ctrl.Result{},
		},
		{
			name:       "matching pods, evaluator returns Stop",
			deployment: makeDeploymentWithSelector("dep-uid-1", appLabels),
			pods:       []*corev1.Pod{makePodWithLabels("p1", appLabels)},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Stop}},
			wantResult: ctrl.Result{},
		},
		{
			name:       "non-matching pod labels are ignored",
			deployment: makeDeploymentWithSelector("dep-uid-1", appLabels),
			pods:       []*corev1.Pod{makePodWithLabels("p-other", map[string]string{"app": "other"})},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			wantResult: ctrl.Result{},
		},
		{
			name:       "nil selector returns early without error",
			deployment: &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "my-dep", Namespace: "default"}},
			pods:       []*corev1.Pod{makePodWithLabels("p1", appLabels)},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			wantResult: ctrl.Result{},
		},
		{
			name:           "Snapshot error is propagated",
			deployment:     makeDeploymentWithSelector("dep-uid-1", appLabels),
			pods:           []*corev1.Pod{makePodWithLabels("p1", appLabels)},
			snapshotErr:    errors.New("boom"),
			wantResult:     ctrl.Result{},
			wantErrPresent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)
			_ = corev1.AddToScheme(s)
			_ = operatorv1alpha2.AddToScheme(s)

			objs := []client.Object{tt.deployment}
			for _, p := range tt.pods {
				objs = append(objs, p)
			}
			objs = append(objs, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   "default",
					Labels: map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
				},
			})

			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(objs...).
				Build()

			r := &DeploymentReconciler{
				Client:     fakeClient,
				Scheme:     s,
				Evaluators: fakeSource{evs: tt.evaluators, err: tt.snapshotErr},
				Recorder:   record.NewFakeRecorder(10),
			}

			got, err := r.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: "my-dep", Namespace: "default"},
			})
			if (err != nil) != tt.wantErrPresent {
				t.Errorf("Reconcile() error = %v, wantErrPresent %v", err, tt.wantErrPresent)
			}
			if got != tt.wantResult {
				t.Errorf("Reconcile() result = %v, want %v", got, tt.wantResult)
			}
		})
	}
}

func TestDeploymentReconcile_NotFound(t *testing.T) {
	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	_ = operatorv1alpha2.AddToScheme(s)

	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()
	r := &DeploymentReconciler{Client: fakeClient, Scheme: s, Recorder: record.NewFakeRecorder(10)}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"},
	})
	if err != nil {
		t.Errorf("expected nil error for not-found Deployment, got %v", err)
	}
}
