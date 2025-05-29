//go:build experimental

package controllers

import (
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
)

func (r *IstioReconciler) validate(istioCR *operatorv1alpha2.Istio) describederrors.DescribedError {
	// when validation is handled in experimental flavour this function
	// does nothing, as validation is only needed for productive environment
	return nil
}
