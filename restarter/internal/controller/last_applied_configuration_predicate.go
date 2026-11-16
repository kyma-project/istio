package controller

import (
	"github.com/kyma-project/istio/operator/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// LastAppliedConfigurationChangedPredicate reacts to switch of the Istio CR
// last applied config annotation.
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
