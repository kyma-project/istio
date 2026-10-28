package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func enqueueWorkloadsOnIstioChange(
	c client.Client,
	newObjectList func() client.ObjectList,
	objectListToClientObjects func(client.ObjectList) []client.Object,
	keep func(client.Object) bool,
) func(context.Context, client.Object) []reconcile.Request {
	return func(ctx context.Context, _ client.Object) []reconcile.Request {
		l := logf.FromContext(ctx)
		list := newObjectList()
		if err := c.List(ctx, list); err != nil {
			l.Error(err, "Failed to list workloads for Istio CR change")
			return nil
		}

		objs := objectListToClientObjects(list)
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
