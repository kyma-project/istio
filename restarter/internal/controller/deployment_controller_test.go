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
	"github.com/kyma-project/istio/operator/restarter/internal/evaluation"
	"github.com/kyma-project/istio/operator/restarter/internal/restart"
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

// TestDeploymentReconcile_GatewayBranch verifies that Reconcile routes an
// istio-ingressgateway Deployment through the gateway path and leaves a
// normal injected Deployment on the sidecar path.
func TestDeploymentReconcile_GatewayBranch(t *testing.T) {
	const gwFP = "gw-fp-xyz"

	t.Run("gateway Deployment with FP mismatch is restarted via gateway path", func(t *testing.T) {
		s := runtime.NewScheme()
		_ = appsv1.AddToScheme(s)
		_ = corev1.AddToScheme(s)
		_ = operatorv1alpha2.AddToScheme(s)

		dep := ingressGatewayDeployment(nil, true)
		pod := gatewayPod("gw-pod-1")

		objs := []client.Object{dep, pod}
		c := fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()

		src := fakeSource{
			gatewayEvs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:  gwFP,
		}
		r := &DeploymentReconciler{
			Client:     c,
			Scheme:     s,
			Evaluators: src,
			Recorder:   record.NewFakeRecorder(10),
		}

		_, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      defaultIstioGatewayName,
				Namespace: defaultIstioGatewayNamespace,
			},
		})
		if err != nil {
			t.Fatalf("Reconcile() error = %v", err)
		}

		got := &appsv1.Deployment{}
		if err := c.Get(context.Background(), types.NamespacedName{
			Name:      defaultIstioGatewayName,
			Namespace: defaultIstioGatewayNamespace,
		}, got); err != nil {
			t.Fatalf("get deployment: %v", err)
		}
		if got.Spec.Template.Annotations[restart.HashAnnotation] != gwFP {
			t.Errorf("HashAnnotation = %q, want %q (gateway path not taken)",
				got.Spec.Template.Annotations[restart.HashAnnotation], gwFP)
		}
		assertRestartAnnotationRecent(t, got.Spec.Template.Annotations[restart.RestartAnnotation])
	})

	t.Run("normal injected Deployment is not affected by gatewayEvs", func(t *testing.T) {
		s := runtime.NewScheme()
		_ = appsv1.AddToScheme(s)
		_ = corev1.AddToScheme(s)
		_ = operatorv1alpha2.AddToScheme(s)

		appLabels := map[string]string{"app": "myapp"}
		dep := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "my-dep", Namespace: "default", UID: "dep-uid"},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{MatchLabels: appLabels},
				Replicas: func() *int32 { ; return new(int32(1)) }(),
			},
			Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
		}
		pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Labels: appLabels}}
		ns := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "default",
				Labels: map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
			},
		}

		objs := []client.Object{dep, pod, ns}
		c := fake.NewClientBuilder().WithScheme(s).WithObjects(objs...).Build()

		src := fakeSource{
			// gatewayEvs votes Restart, but evs (sidecar path) votes Continue.
			// The non-gateway Deployment must not be restarted.
			evs:        []evaluation.Evaluator{&fakeEvaluator{evaluation.Continue}},
			gatewayEvs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:  gwFP,
		}
		r := &DeploymentReconciler{
			Client:     c,
			Scheme:     s,
			Evaluators: src,
			Recorder:   record.NewFakeRecorder(10),
		}

		_, err := r.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{Name: "my-dep", Namespace: "default"},
		})
		if err != nil {
			t.Fatalf("Reconcile() error = %v", err)
		}

		got := &appsv1.Deployment{}
		if err := c.Get(context.Background(), types.NamespacedName{Name: "my-dep", Namespace: "default"}, got); err != nil {
			t.Fatalf("get deployment: %v", err)
		}
		if got.Spec.Template.Annotations[restart.RestartAnnotation] != "" {
			t.Errorf("non-gateway Deployment was restarted via gateway path: restartedAt = %q",
				got.Spec.Template.Annotations[restart.RestartAnnotation])
		}
	})
}
