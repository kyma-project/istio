//go:build !experimental

package controllers

import (
	"github.com/pkg/errors"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/describederrors"
)

func (r *IstioReconciler) validate(istioCR *operatorv1alpha2.Istio) describederrors.DescribedError {
	if istioCR.Spec.Experimental != nil {
		// user has experimental field applied in their CR
		// return error with description
		r.log.Info("Experimental features are not supported in this image flavour")
		return describederrors.NewDescribedError(errors.New("istio CR contains experimental feature"), "Experimental features are not supported in this image flavour").
			SetWarning().
			SetCondition(false)
	}
	return nil
}
