package istio

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coreos/go-semver/semver"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/common"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ingressgatewayNamespace      string = "istio-system"
	ingressgatewayDeploymentName string = "istio-ingressgateway"
)

type AppliedConfig struct {
	operatorv1alpha1.IstioSpec
	IstioTag string
}

// shouldDelete returns true when Istio should be deleted
func shouldDelete(istio operatorv1alpha1.Istio) bool {
	return !istio.DeletionTimestamp.IsZero()
}

// shouldInstall returns true when Istio should be installed
func shouldInstall(istio operatorv1alpha1.Istio, istioTag string) (shouldInstall bool, err error) {
	if shouldDelete(istio) {
		return false, nil
	}

	lastAppliedConfigAnnotation, ok := istio.Annotations[LastAppliedConfiguration]
	if !ok {
		return true, nil
	}

	var lastAppliedConfig AppliedConfig
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
func UpdateLastAppliedConfiguration(istio operatorv1alpha1.Istio, istioTag string) (operatorv1alpha1.Istio, error) {
	if len(istio.Annotations) == 0 {
		istio.Annotations = map[string]string{}
	}

	newAppliedConfig := AppliedConfig{
		IstioSpec: istio.Spec,
		IstioTag:  istioTag,
	}

	config, err := json.Marshal(newAppliedConfig)
	if err != nil {
		return operatorv1alpha1.Istio{}, err
	}

	istio.Annotations[LastAppliedConfiguration] = string(config)

	return istio, nil
}

func GetLastAppliedConfiguration(istio operatorv1alpha1.Istio) (AppliedConfig, error) {
	lastAppliedConfig := AppliedConfig{}
	if len(istio.Annotations) == 0 {
		return lastAppliedConfig, nil
	}

	if lastAppliedAnnotation, found := istio.Annotations[LastAppliedConfiguration]; found {
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

func amongOneMinor(current, target semver.Version) bool {
	return current.Minor == target.Minor || current.Minor-target.Minor == -1 || current.Minor-target.Minor == 1
}

func restartIngressGateway(ctx context.Context, k8sClient client.Client) error {
	deployment := appsv1.Deployment{}
	err := k8sClient.Get(ctx, types.NamespacedName{Namespace: ingressgatewayNamespace, Name: ingressgatewayDeploymentName}, &deployment)
	if err != nil {
		return err
	}
	deployment.Spec.Template.Annotations = common.AddRestartAnnotation(deployment.Spec.Template.Annotations)
	return k8sClient.Update(ctx, &deployment)
}

func ingressGatewayNeedsRestart(istioCR operatorv1alpha1.Istio) (bool, error) {
	lastAppliedConfig, err := GetLastAppliedConfiguration(istioCR)
	if err != nil {
		return false, err
	}

	isNewNotNil := (istioCR.Spec.Config.NumTrustedProxies != nil)
	isOldNotNil := (lastAppliedConfig.IstioSpec.Config.NumTrustedProxies != nil)
	if isNewNotNil && isOldNotNil && *istioCR.Spec.Config.NumTrustedProxies != *lastAppliedConfig.IstioSpec.Config.NumTrustedProxies {
		return true, nil
	}
	if isNewNotNil != isOldNotNil {
		return true, nil
	}

	return false, nil
}
