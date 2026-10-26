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

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// enqueueWorkloadsOnIstioChange returns a map function that, when the watched
// Istio CR changes, enqueues every workload of the type produced by newList.
// The Istio CR carries the evaluator config (sidecar image, proxy resources,
// features); a change to it must re-evaluate all injected workloads, which
// their own object events would never trigger on their own.
//
// keep filters which listed objects are enqueued (e.g. ownerless ReplicaSets);
// pass nil to enqueue every object.
func enqueueWorkloadsOnIstioChange(
	c client.Client,
	newList func() client.ObjectList,
	items func(client.ObjectList) []client.Object,
	keep func(client.Object) bool,
) func(context.Context, client.Object) []reconcile.Request {
	return func(ctx context.Context, _ client.Object) []reconcile.Request {
		l := logf.FromContext(ctx)
		list := newList()
		if err := c.List(ctx, list); err != nil {
			l.Error(err, "Failed to list workloads for Istio CR change")
			return nil
		}

		objs := items(list)
		reqs := make([]reconcile.Request, 0, len(objs))
		for _, obj := range objs {
			if keep != nil && !keep(obj) {
				continue
			}
			reqs = append(reqs, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				},
			})
		}
		return reqs
	}
}
