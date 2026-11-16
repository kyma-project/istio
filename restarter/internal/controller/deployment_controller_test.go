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
	const gwFP = "gw-fp-xyz"

	normalInjectedDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "my-dep", Namespace: "default", UID: "dep-uid"},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: appLabels},
			Replicas: new(int32(1)),
		},
		Status: appsv1.DeploymentStatus{Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1},
	}

	tests := []struct {
		name           string
		deployment     *appsv1.Deployment
		extraObjs      []client.Object
		pods           []*corev1.Pod
		evaluators     []evaluation.Evaluator
		gatewayEvs     []evaluation.Evaluator
		gatewayFP      string
		snapshotErr    error
		missing        bool
		skipNamespace  bool
		reconcileName  types.NamespacedName
		wantResult     ctrl.Result
		wantErrPresent bool
		check          func(t *testing.T, c client.Client)
	}{
		{
			name:       "no pods, no evaluators → Continue, no error",
			deployment: makeDeploymentWithSelector("dep-uid-1", appLabels),
			pods:       nil,
			evaluators: nil,
			wantResult: ctrl.Result{},
		},
		{
			name:       "missing Deployment → nil error, empty result",
			missing:    true,
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
		{
			name:          "gateway Deployment with FP mismatch is restarted via gateway path",
			deployment:    ingressGatewayDeployment(nil, true),
			extraObjs:     []client.Object{gatewayPod("gw-pod-1")},
			gatewayEvs:    []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:     gwFP,
			skipNamespace: true,
			reconcileName: types.NamespacedName{Name: defaultIstioGatewayName, Namespace: defaultIstioGatewayNamespace},
			wantResult:    ctrl.Result{},
			check: func(t *testing.T, c client.Client) {
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
			},
		},
		{
			// gatewayEvs votes Restart, but evs (sidecar path) votes Continue.
			// A non-gateway Deployment must stay on the sidecar path and not be restarted.
			name:       "normal injected Deployment is not affected by gatewayEvs",
			deployment: normalInjectedDeployment,
			pods:       []*corev1.Pod{makePodWithLabels("p", appLabels)},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Continue}},
			gatewayEvs: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			gatewayFP:  gwFP,
			wantResult: ctrl.Result{},
			check: func(t *testing.T, c client.Client) {
				got := &appsv1.Deployment{}
				if err := c.Get(context.Background(), types.NamespacedName{Name: "my-dep", Namespace: "default"}, got); err != nil {
					t.Fatalf("get deployment: %v", err)
				}
				if got.Spec.Template.Annotations[restart.RestartAnnotation] != "" {
					t.Errorf("non-gateway Deployment was restarted via gateway path: restartedAt = %q",
						got.Spec.Template.Annotations[restart.RestartAnnotation])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)
			_ = corev1.AddToScheme(s)
			_ = operatorv1alpha2.AddToScheme(s)

			objs := []client.Object{}
			if tt.deployment != nil {
				objs = append(objs, tt.deployment)
			}
			for _, p := range tt.pods {
				objs = append(objs, p)
			}
			objs = append(objs, tt.extraObjs...)
			if !tt.skipNamespace {
				objs = append(objs, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "default",
						Labels: map[string]string{evaluation.InjectionLabel: evaluation.InjectEnabled},
					},
				})
			}

			fakeClient := fake.NewClientBuilder().
				WithScheme(s).
				WithObjects(objs...).
				Build()

			r := &DeploymentReconciler{
				Client: fakeClient,
				Scheme: s,
				Evaluators: fakeSource{
					evs:        tt.evaluators,
					err:        tt.snapshotErr,
					gatewayEvs: tt.gatewayEvs,
					gatewayFP:  tt.gatewayFP,
				},
				Recorder: record.NewFakeRecorder(10),
			}

			reqName := tt.reconcileName
			if reqName == (types.NamespacedName{}) {
				name := "my-dep"
				if tt.missing {
					name = "missing"
				}
				reqName = types.NamespacedName{Name: name, Namespace: "default"}
			}

			got, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: reqName})
			if (err != nil) != tt.wantErrPresent {
				t.Errorf("Reconcile() error = %v, wantErrPresent %v", err, tt.wantErrPresent)
			}
			if got != tt.wantResult {
				t.Errorf("Reconcile() result = %v, want %v", got, tt.wantResult)
			}
			if tt.check != nil {
				tt.check(t, fakeClient)
			}
		})
	}
}
