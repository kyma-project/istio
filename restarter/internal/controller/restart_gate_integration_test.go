package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
)

// TestReconcile_IstioCRStateIgnored proves the Istio CR status is not consulted
// before performing a restart. A Restart decision proceeds regardless of whether
// the CR is Processing, Deleting, or absent.
func TestReconcile_IstioCRStateIgnored(t *testing.T) {
	withFixedNow(t)

	appLabels := map[string]string{"app": "myapp"}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: "w", Namespace: "default"}}

	istioCR := func(state operatorv1alpha2.State) *operatorv1alpha2.Istio {
		return &operatorv1alpha2.Istio{
			ObjectMeta: metav1.ObjectMeta{Name: "default", Namespace: "kyma-system"},
			Status:     operatorv1alpha2.IstioStatus{State: state},
		}
	}

	newScheme := func() *runtime.Scheme {
		s := runtime.NewScheme()
		if err := appsv1.AddToScheme(s); err != nil {
			t.Fatalf("add apps scheme: %v", err)
		}
		if err := corev1.AddToScheme(s); err != nil {
			t.Fatalf("add core scheme: %v", err)
		}
		if err := operatorv1alpha2.AddToScheme(s); err != nil {
			t.Fatalf("add istio scheme: %v", err)
		}
		return s
	}

	tests := []struct {
		name string
		cr   *operatorv1alpha2.Istio
	}{
		{name: "Processing CR does not block restart", cr: istioCR(operatorv1alpha2.Processing)},
		{name: "Deleting CR does not block restart", cr: istioCR(operatorv1alpha2.Deleting)},
		{name: "absent CR does not block restart", cr: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{Name: "w", Namespace: "default", UID: "dep"},
				Spec: appsv1.DeploymentSpec{
					Selector: &metav1.LabelSelector{MatchLabels: appLabels},
					Replicas: ptr.To(int32(1)),
				},
				Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
			}
			pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: "p", Namespace: "default", Labels: appLabels}}

			objs := []interface{}{dep, pod, injectedNamespace()}
			b := fake.NewClientBuilder().WithScheme(newScheme())
			if tt.cr != nil {
				b = b.WithObjects(dep, pod, injectedNamespace(), tt.cr)
			} else {
				b = b.WithObjects(dep, pod, injectedNamespace())
			}
			_ = objs
			c := b.Build()

			r := &DeploymentReconciler{
				Client:     c,
				Evaluators: fakeSource{evs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}}},
				Recorder:   record.NewFakeRecorder(10),
			}
			res, err := r.Reconcile(context.Background(), req)
			if err != nil {
				t.Fatalf("reconcile: %v", err)
			}
			if res != (ctrl.Result{}) {
				t.Errorf("result = %v, want empty (no requeue from CR status)", res)
			}

			got := &appsv1.Deployment{}
			if err := c.Get(context.Background(), req.NamespacedName, got); err != nil {
				t.Fatalf("get deployment: %v", err)
			}
			if got.Spec.Template.Annotations[RestartAnnotation] != fixedTime {
				t.Errorf("restart annotation = %q, want %q (restart should proceed regardless of CR state)",
					got.Spec.Template.Annotations[RestartAnnotation], fixedTime)
			}
		})
	}
}
