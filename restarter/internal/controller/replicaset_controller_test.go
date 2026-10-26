/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"k8s.io/client-go/tools/record"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/workload-controller/internal/evaluation"
)

func TestReplicaSetReconcile(t *testing.T) {
	rsUID := types.UID("rs-uid-1")

	makePod := func(name string, ownerUID types.UID) *corev1.Pod {
		return &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "default",
				OwnerReferences: []metav1.OwnerReference{
					{Kind: "ReplicaSet", UID: ownerUID},
				},
			},
		}
	}

	makeRS := func(uid types.UID) *appsv1.ReplicaSet {
		return &appsv1.ReplicaSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-rs",
				Namespace: "default",
				UID:       uid,
			},
		}
	}

	tests := []struct {
		name           string
		pods           []*corev1.Pod
		evaluators     []evaluation.Evaluator
		snapshotErr    error
		wantResult     ctrl.Result
		wantErrPresent bool
	}{
		{
			name:       "no pods, no evaluators → Continue, no error",
			pods:       nil,
			evaluators: nil,
			wantResult: ctrl.Result{},
		},
		{
			name:       "pods owned by this RS, evaluator returns Continue",
			pods:       []*corev1.Pod{makePod("p1", rsUID)},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Continue}},
			wantResult: ctrl.Result{},
		},
		{
			name:       "pods owned by this RS, evaluator returns Restart",
			pods:       []*corev1.Pod{makePod("p1", rsUID)},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Restart}},
			wantResult: ctrl.Result{},
		},
		{
			name:       "pods owned by different RS are ignored",
			pods:       []*corev1.Pod{makePod("p-other", "other-uid")},
			evaluators: []evaluation.Evaluator{&fakeEvaluator{evaluation.Stop}},
			wantResult: ctrl.Result{},
		},
		{
			name:           "Snapshot error is propagated",
			pods:           []*corev1.Pod{makePod("p1", rsUID)},
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

			objs := []client.Object{makeRS(rsUID)}
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
				WithIndex(&corev1.Pod{}, replicaSetPodOwnerKey, func(obj client.Object) []string {
					pod, ok := obj.(*corev1.Pod)
					if !ok {
						return nil
					}
					for _, ref := range pod.OwnerReferences {
						if ref.Kind == "ReplicaSet" {
							return []string{string(ref.UID)}
						}
					}
					return nil
				}).
				Build()

			r := &ReplicaSetReconciler{
				Client:     fakeClient,
				Scheme:     s,
				Evaluators: fakeSource{evs: tt.evaluators, err: tt.snapshotErr},
				Recorder:   record.NewFakeRecorder(10),
			}

			got, err := r.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{Name: "my-rs", Namespace: "default"},
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

func TestReplicaSetReconcile_NotFound(t *testing.T) {
	s := runtime.NewScheme()
	_ = appsv1.AddToScheme(s)
	_ = corev1.AddToScheme(s)
	_ = operatorv1alpha2.AddToScheme(s)

	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()
	r := &ReplicaSetReconciler{Client: fakeClient, Scheme: s}

	_, err := r.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "missing", Namespace: "default"},
	})
	if err != nil {
		t.Errorf("expected nil error for not-found RS, got %v", err)
	}
}

// fakeEvaluator is shared across controller tests in this package.
type fakeEvaluator struct{ d evaluation.Decision }

func (f *fakeEvaluator) Name() string                           { return "fakeEvaluator" }
func (f *fakeEvaluator) Evaluate(_ client.Object) evaluation.Decision { return f.d }
