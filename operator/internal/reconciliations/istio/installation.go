package istio

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
)

type Installation struct {
	Client         IstioClient
	IstioVersion   string
	IstioImageBase string
}

const (
	LastAppliedConfiguration string = "operator.kyma-project.io/lastAppliedConfiguration"
)

// Reconcile runs Istio installation with merged Istio Operator manifest file when the trigger requires an installation.
func (i *Installation) Reconcile(istioCR *operatorv1alpha1.Istio) error {
	mergedIstioOperatorPath, err := merge(istioCR, i.Client.defaultIstioOperatorPath, i.Client.workingDir, TemplateData{IstioVersion: i.IstioVersion, IstioImageBase: i.IstioImageBase})
	if err != nil {
		return err
	}

	err = i.Client.Install(mergedIstioOperatorPath)
	if err != nil {
		return err
	}

	return nil
}
