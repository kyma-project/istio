package istio

import (
	"encoding/json"
	"fmt"
	"github.com/coreos/go-semver/semver"

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

func (r IstioCRChange) requireInstall() bool {
	return r == Create || r&VersionUpdate > 0 || r&ConfigurationUpdate > 0
}

func (r IstioCRChange) requireIstioDeletion() bool {
	return r == Delete
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

	trigger = NoChange

	var lastAppliedConfig appliedConfig
	err = json.Unmarshal([]byte(lastAppliedConfigAnnotation), &lastAppliedConfig)
	if err != nil {
		return trigger, err
	}

	err = CheckIstioVersion(lastAppliedConfig.IstioTag, istioTag)
	if err != nil {
		return trigger, err
	}

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

func CheckIstioVersion(currentIstioVersionString, targetIstioVersionString string) error {
	currentIstioVersion, err := semver.NewVersion(currentIstioVersionString)
	if err != nil {
		return err
	}
	targetIstioVersion, err := semver.NewVersion(targetIstioVersionString)
	if err != nil {
		return err
	}

	if targetIstioVersion.LessThan(*currentIstioVersion) {
		return fmt.Errorf("target Istio version (%s) is lower than current version (%s) - downgrade not supported",
			targetIstioVersion.String(), currentIstioVersion.String())
	}
	if !(currentIstioVersion.Major == targetIstioVersion.Major) {
		return fmt.Errorf("target Istio version (%s) is different than current Istio version (%s) - major version upgrade is not supported", targetIstioVersion.String(), currentIstioVersion.String())
	}
	if !amongOneMinor(*currentIstioVersion, *targetIstioVersion) {
		return fmt.Errorf("target Istio version (%s) is higher than current Istio version (%s) - the difference between versions exceed one minor version", targetIstioVersion.String(), currentIstioVersion.String())
	}

	return nil
}

func amongOneMinor(current, target semver.Version) bool {
	return current.Minor == target.Minor || current.Minor-target.Minor == -1 || current.Minor-target.Minor == 1
}
