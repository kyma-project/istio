package controller

import (
	"github.com/kyma-project/istio/operator/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LastAppliedConfigurationChangedPredicate enqueues an Istio CR update only when the
// lastAppliedConfiguration annotation is newly set or its value changes. The operator
// writes this annotation at the end of a successful reconcile, so reconciliation (and
// the restart step it drives) is enqueued after the applied configuration is persisted
// rather than on the raw spec edit. Create/Delete/Generic keep the default (true) so
// initial reconcile and finalizer handling are unaffected.
type LastAppliedConfigurationChangedPredicate struct {
	predicate.Funcs
}

func (LastAppliedConfigurationChangedPredicate) Update(e event.TypedUpdateEvent[client.Object]) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}
	return e.ObjectOld.GetAnnotations()[labels.LastAppliedConfiguration] !=
		e.ObjectNew.GetAnnotations()[labels.LastAppliedConfiguration]
}
