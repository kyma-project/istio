package istio

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
)

type Installation struct {
	Client IstioClient
}

// Reconcile setup configuration and runs an Istio installation with merged Istio Operator manifest file.
func (i *Installation) Reconcile(istioCR *operatorv1alpha1.Istio) error {
	mergedIstioOperator, err := merge(istioCR)
	if err != nil {
		return err
	}
	return i.Client.Install(mergedIstioOperator)
}
