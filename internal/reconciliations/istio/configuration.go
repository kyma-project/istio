package istio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coreos/go-semver/semver"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	"github.com/kyma-project/istio/operator/internal/restarter"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type appliedConfig struct {
	operatorv1alpha2.IstioSpec
	IstioTag string
}

// shouldDelete returns true when Istio should be deleted
func shouldDelete(istio *operatorv1alpha2.Istio) bool {
	return !istio.DeletionTimestamp.IsZero()
}

// shouldInstall returns true when Istio should be installed
func shouldInstall(istio *operatorv1alpha2.Istio, istioImageVersion istiooperator.IstioImageVersion) (shouldInstall bool, err error) {
	if shouldDelete(istio) {
		return false, nil
	}

	lastAppliedConfigAnnotation, ok := istio.Annotations[LastAppliedConfiguration]
	if !ok {
		return true, nil
	}

	var lastAppliedConfig appliedConfig
	if err := json.Unmarshal([]byte(lastAppliedConfigAnnotation), &lastAppliedConfig); err != nil {
		return false, err
	}

	if err := CheckIstioVersion(lastAppliedConfig.IstioTag, istioImageVersion.Tag()); err != nil {
		return false, err
	}

	return true, nil
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

	istioCR.Annotations[LastAppliedConfiguration] = string(config)
	return nil
}

func getLastAppliedConfiguration(istioCR *operatorv1alpha2.Istio) (appliedConfig, error) {
	lastAppliedConfig := appliedConfig{}
	if len(istioCR.Annotations) == 0 {
		return lastAppliedConfig, nil
	}

	if lastAppliedAnnotation, found := istioCR.Annotations[LastAppliedConfiguration]; found {
		err := json.Unmarshal([]byte(lastAppliedAnnotation), &lastAppliedConfig)
		if err != nil {
			return lastAppliedConfig, err
		}
	}

	return lastAppliedConfig, nil
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

func restartIngressGatewayIfNeeded(ctx context.Context, k8sClient client.Client, istioCR *operatorv1alpha2.Istio) error {
	mustRestart := false

	lastAppliedConfig, err := getLastAppliedConfiguration(istioCR)
	if err != nil {
		return err
	}

	isNewNotNil := (istioCR.Spec.Config.NumTrustedProxies != nil)
	isOldNotNil := (lastAppliedConfig.IstioSpec.Config.NumTrustedProxies != nil)
	if isNewNotNil && isOldNotNil && *istioCR.Spec.Config.NumTrustedProxies != *lastAppliedConfig.IstioSpec.Config.NumTrustedProxies {
		mustRestart = true
	} else if isNewNotNil != isOldNotNil {
		mustRestart = true
	}

	if mustRestart {
		err := restarter.RestartIngressGateway(ctx, k8sClient)
		if err != nil {
			return err
		}
	}
	return nil
}

func amongOneMinor(current, target semver.Version) bool {
	return current.Minor == target.Minor || current.Minor-target.Minor == -1 || current.Minor-target.Minor == 1
}
