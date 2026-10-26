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
	"sort"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func deploymentItems(list client.ObjectList) []client.Object {
	items := list.(*appsv1.DeploymentList).Items
	objs := make([]client.Object, len(items))
	for i := range items {
		objs[i] = &items[i]
	}
	return objs
}

func replicaSetItems(list client.ObjectList) []client.Object {
	items := list.(*appsv1.ReplicaSetList).Items
	objs := make([]client.Object, len(items))
	for i := range items {
		objs[i] = &items[i]
	}
	return objs
}

func dep(name, ns string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: ns}}
}

func rs(name, ns string, owned bool) *appsv1.ReplicaSet {
	meta := metav1.ObjectMeta{Name: name, Namespace: ns}
	if owned {
		meta.OwnerReferences = []metav1.OwnerReference{{Kind: "Deployment", APIVersion: "apps/v1", UID: types.UID("owner")}}
	}
	return &appsv1.ReplicaSet{ObjectMeta: meta}
}

func TestEnqueueWorkloadsOnIstioChange(t *testing.T) {
	ownerless := func(obj client.Object) bool { return len(obj.GetOwnerReferences()) == 0 }

	tests := []struct {
		name     string
		objects  []client.Object
		newList  func() client.ObjectList
		items    func(client.ObjectList) []client.Object
		keep     func(client.Object) bool
		listErr  error
		wantReqs []reconcile.Request
	}{
		{
			name:    "enqueues every deployment",
			objects: []client.Object{dep("a", "ns1"), dep("b", "ns2")},
			newList: func() client.ObjectList { return &appsv1.DeploymentList{} },
			items:   deploymentItems,
			keep:    nil,
			wantReqs: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "a", Namespace: "ns1"}},
				{NamespacedName: types.NamespacedName{Name: "b", Namespace: "ns2"}},
			},
		},
		{
			name:     "no workloads enqueues nothing",
			objects:  nil,
			newList:  func() client.ObjectList { return &appsv1.DeploymentList{} },
			items:    deploymentItems,
			keep:     nil,
			wantReqs: []reconcile.Request{},
		},
		{
			name:    "keep filter drops owned replicasets",
			objects: []client.Object{rs("own-a", "ns1", false), rs("owned-b", "ns1", true), rs("own-c", "ns2", false)},
			newList: func() client.ObjectList { return &appsv1.ReplicaSetList{} },
			items:   replicaSetItems,
			keep:    ownerless,
			wantReqs: []reconcile.Request{
				{NamespacedName: types.NamespacedName{Name: "own-a", Namespace: "ns1"}},
				{NamespacedName: types.NamespacedName{Name: "own-c", Namespace: "ns2"}},
			},
		},
		{
			name:     "list error enqueues nothing",
			objects:  []client.Object{dep("a", "ns1")},
			newList:  func() client.ObjectList { return &appsv1.DeploymentList{} },
			items:    deploymentItems,
			keep:     nil,
			listErr:  errors.New("boom"),
			wantReqs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := runtime.NewScheme()
			_ = appsv1.AddToScheme(s)
			_ = corev1.AddToScheme(s)

			builder := fake.NewClientBuilder().WithScheme(s).WithObjects(tt.objects...)
			if tt.listErr != nil {
				builder = builder.WithInterceptorFuncs(interceptor.Funcs{
					List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
						return tt.listErr
					},
				})
			}
			c := builder.Build()

			mapFn := enqueueWorkloadsOnIstioChange(c, tt.newList, tt.items, tt.keep)
			got := mapFn(context.Background(), nil)

			sortRequests(got)
			sortRequests(tt.wantReqs)
			if !equalRequests(got, tt.wantReqs) {
				t.Errorf("mapFn() = %v, want %v", got, tt.wantReqs)
			}
		})
	}
}

func sortRequests(reqs []reconcile.Request) {
	sort.Slice(reqs, func(i, j int) bool {
		if reqs[i].Namespace != reqs[j].Namespace {
			return reqs[i].Namespace < reqs[j].Namespace
		}
		return reqs[i].Name < reqs[j].Name
	})
}

func equalRequests(a, b []reconcile.Request) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
