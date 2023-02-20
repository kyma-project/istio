package istio

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
)

type Uninstallation struct {
	Client         IstioClient
	IstioVersion   string
	IstioImageBase string
}

// Reconcile runs Istio uninstallation.
func (i *Uninstallation) Reconcile(istioCR *operatorv1alpha1.Istio) error {
	err := i.Client.Uninstall()
	if err != nil {
		return err
	}

	return nil
}
