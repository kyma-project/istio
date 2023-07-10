package istio

import (
	"encoding/json"
	"fmt"
	"github.com/coreos/go-semver/semver"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
)

type appliedConfig struct {
	operatorv1alpha1.IstioSpec
	IstioTag string
}

// shouldDelete returns true when Istio should be deleted
func shouldDelete(istioCR operatorv1alpha1.Istio) bool {
	return !istioCR.DeletionTimestamp.IsZero()
}

// shouldInstall returns true when Istio should be installed
func shouldInstall(istioCR operatorv1alpha1.Istio, istioTag string) (shouldInstall bool, err error) {
	if shouldDelete(istioCR) {
		return false, nil
	}

	lastAppliedConfigAnnotation, ok := istioCR.Annotations[LastAppliedConfiguration]
	if !ok {
		return true, nil
	}

	var lastAppliedConfig appliedConfig
	if err := json.Unmarshal([]byte(lastAppliedConfigAnnotation), &lastAppliedConfig); err != nil {
		return false, err
	}

	if err := CheckIstioVersion(lastAppliedConfig.IstioTag, istioTag); err != nil {
		return false, err
	}

	return true, nil
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
	if currentIstioVersion.Major != targetIstioVersion.Major {
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
