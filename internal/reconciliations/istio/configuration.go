package istio

import (
	"encoding/json"
	"fmt"
	"github.com/kyma-project/istio/operator/pkg/labels"

	"github.com/coreos/go-semver/semver"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
)

type appliedConfig struct {
	operatorv1alpha2.IstioSpec
	IstioTag string
}

// isIstioInstalled returns true when Istio CR has LastAppliedConfiguration annotation
func isIstioInstalled(istio *operatorv1alpha2.Istio) bool {
	_, ok := istio.Annotations[labels.LastAppliedConfiguration]
	return ok
}

// UpdateLastAppliedConfiguration annotates the passed CR with LastAppliedConfiguration, which holds information about last applied
// IstioCR spec and IstioTag (IstioVersion-IstioImageBase)
func UpdateLastAppliedConfiguration(istioCR *operatorv1alpha2.Istio, istioTag string) error {
	if len(istioCR.Annotations) == 0 {
		istioCR.Annotations = map[string]string{}
	}

	newAppliedConfig := appliedConfig{
		IstioSpec: istioCR.Spec,
		IstioTag:  istioTag,
	}

	config, err := json.Marshal(newAppliedConfig)
	if err != nil {
		return err
	}

	istioCR.Annotations[labels.LastAppliedConfiguration] = string(config)
	return nil
}

func getLastAppliedConfiguration(istioCR *operatorv1alpha2.Istio) (appliedConfig, error) {
	lastAppliedConfig := appliedConfig{}
	if len(istioCR.Annotations) == 0 {
		return lastAppliedConfig, nil
	}

	if lastAppliedAnnotation, found := istioCR.Annotations[labels.LastAppliedConfiguration]; found {
		err := json.Unmarshal([]byte(lastAppliedAnnotation), &lastAppliedConfig)
		if err != nil {
			return lastAppliedConfig, err
		}
	}

	return lastAppliedConfig, nil
}

func checkIstioVersionUpdate(currentIstioVersionString, targetIstioVersionString string) error {
	currentIstioVersion, err := semver.NewVersion(currentIstioVersionString)
	if err != nil {
		return err
	}
	targetIstioVersion, err := semver.NewVersion(targetIstioVersionString)
	if err != nil {
		return err
	}

	// We need to compare this separately, because semver library does not support comparing versions by ignoring pre-release versions. But only a changed image type must not be considered
	// as a change of the Istio version.
	if currentIstioVersion.Major == targetIstioVersion.Major && currentIstioVersion.Minor == targetIstioVersion.Minor && currentIstioVersion.Patch == targetIstioVersion.Patch {
		return nil
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
