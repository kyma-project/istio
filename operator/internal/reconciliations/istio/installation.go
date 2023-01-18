package istio

import (
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	istiolog "istio.io/pkg/log"
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
	if i.Client.istioLogOptions == nil {
		i.Client.istioLogOptions = istiolog.DefaultOptions()
		i.Client.istioLogOptions.SetOutputLevel("validation", istiolog.ErrorLevel)
		i.Client.istioLogOptions.SetOutputLevel("processing", istiolog.ErrorLevel)
		i.Client.istioLogOptions.SetOutputLevel("analysis", istiolog.WarnLevel)
		i.Client.istioLogOptions.SetOutputLevel("installer", istiolog.WarnLevel)
		i.Client.istioLogOptions.SetOutputLevel("translator", istiolog.WarnLevel)
		i.Client.istioLogOptions.SetOutputLevel("adsc", istiolog.WarnLevel)
		i.Client.istioLogOptions.SetOutputLevel("default", istiolog.WarnLevel)
		i.Client.istioLogOptions.SetOutputLevel("klog", istiolog.WarnLevel)
		i.Client.istioLogOptions.SetOutputLevel("kube", istiolog.ErrorLevel)
	}
	return i.Client.Install(mergedIstioOperator)
}
