package istio

import (
	"encoding/json"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
)

// IstioCRChange represents difference since last reconciliation of IstioCR
type IstioCRChange int

const (
	NoChange            IstioCRChange = 0
	Create              IstioCRChange = 1
	VersionUpdate       IstioCRChange = 2
	ConfigurationUpdate IstioCRChange = 4
	Delete              IstioCRChange = 8
)

type Status struct{}

// TODO rename to shouldIstioReconcile, add delete handling
func (r IstioCRChange) NeedsIstioInstall() bool {
	return r == Create || r&VersionUpdate > 0 || r&ConfigurationUpdate > 0
}

type appliedConfig struct {
	operatorv1alpha1.IstioSpec
	IstioTag string
}

// EvaluateIstioCRChanges returns IstioCRChange that happened since LastAppliedConfiguration
func EvaluateIstioCRChanges(istioCR operatorv1alpha1.Istio, istioTag string) (trigger IstioCRChange, err error) {
	if !istioCR.DeletionTimestamp.IsZero() {
		return Delete, nil
	}

	lastAppliedConfigAnnotation, ok := istioCR.Annotations[LastAppliedConfiguration]
	if !ok {
		return Create, nil
	}

	var lastAppliedConfig appliedConfig
	json.Unmarshal([]byte(lastAppliedConfigAnnotation), &lastAppliedConfig)

	trigger = NoChange
	if lastAppliedConfig.IstioTag != istioTag {
		trigger = trigger | VersionUpdate
	}

	lastAppliedNotNil := lastAppliedConfig.Config.NumTrustedProxies != nil
	newNotNil := istioCR.Spec.Config.NumTrustedProxies != nil

	if lastAppliedNotNil != newNotNil {
		return trigger | ConfigurationUpdate, nil
	}

	if !lastAppliedNotNil {
		return trigger, nil
	}

	if *lastAppliedConfig.Config.NumTrustedProxies != *istioCR.Spec.Config.NumTrustedProxies {
		return trigger | ConfigurationUpdate, nil
	}

	return trigger, nil
}

// UpdateLastAppliedConfiguration annotates the passed CR with LastAppliedConfiguration, which holds information about last applied
// IstioCR spec and IstioTag (IstioVersion-IstioImageBase)
func UpdateLastAppliedConfiguration(cr operatorv1alpha1.Istio, istioTag string) (operatorv1alpha1.Istio, error) {
	if cr.Annotations == nil {
		cr.Annotations = make(map[string]string)
	}

	newAppliedConfig := appliedConfig{
		IstioSpec: cr.Spec,
		IstioTag:  istioTag,
	}

	config, err := json.Marshal(newAppliedConfig)
	if err != nil {
		return operatorv1alpha1.Istio{}, err
	}

	cr.Annotations[LastAppliedConfiguration] = string(config)

	return cr, nil
}
