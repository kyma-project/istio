package controller

import (
	"k8s.io/apimachinery/pkg/api/meta"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
)

func (r *IstioReconciler) shouldSetProcessing(istioCR *operatorv1alpha2.Istio) bool {
	if istioCR.Status.Conditions == nil {
		r.log.Info("Istio CR has no Ready condition yet, setting Processing status", "Istio", istioCR.Name)
		return true
	}

	readyCond := meta.FindStatusCondition(*istioCR.Status.Conditions, "Ready")
	if readyCond == nil {
		r.log.Info("Istio CR has no Ready condition yet, setting Processing status", "Istio", istioCR.Name)
		return true
	}

	if istioCR.Generation > readyCond.ObservedGeneration {
		r.log.Info("Istio CR spec changed, setting Processing status", "Istio", istioCR.Name,
			"generation", istioCR.Generation, "observedGeneration", readyCond.ObservedGeneration)
		return true
	}

	return false
}
