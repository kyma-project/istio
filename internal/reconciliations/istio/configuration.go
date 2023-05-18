package istio

import (
	"encoding/json"
	"fmt"
	"github.com/coreos/go-semver/semver"
	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"reflect"
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

	if nilChange(lastAppliedConfig.Components, istioCR.Spec.Components) {
		return trigger | ConfigurationUpdate, nil
	}

	if lastAppliedConfig.Components != nil {
		trigger |= checkComponentsConfigChange(lastAppliedConfig.Components, istioCR.Spec.Components)
	}

	if nilChange(lastAppliedConfig.Config.NumTrustedProxies, istioCR.Spec.Config.NumTrustedProxies) {
		return trigger | ConfigurationUpdate, nil
	}

	if lastAppliedConfig.Config.NumTrustedProxies == nil {
		return trigger, nil
	}

	if *lastAppliedConfig.Config.NumTrustedProxies != *istioCR.Spec.Config.NumTrustedProxies {
		return trigger | ConfigurationUpdate, nil
	}

	return trigger, nil
}

func checkComponentsConfigChange(components *operatorv1alpha1.Components, components2 *operatorv1alpha1.Components) IstioCRChange {
	if nilChange(components.Pilot, components2.Pilot) || (len(components.IngressGateways) != len(components2.IngressGateways)) {
		return ConfigurationUpdate
	}

	if components.Pilot != nil {
		if checkK8SConfigChange(components.Pilot.K8s, components2.Pilot.K8s) {
			return ConfigurationUpdate
		}
	}

	if len(components.IngressGateways) > 0 {
		for i, ingressGateway := range components.IngressGateways {
			if checkK8SConfigChange(ingressGateway.K8s, components2.IngressGateways[i].K8s) {
				return ConfigurationUpdate
			}
		}
	}

	return NoChange
}

func checkK8SConfigChange(config operatorv1alpha1.KubernetesResourcesConfig, config2 operatorv1alpha1.KubernetesResourcesConfig) bool {
	if nilChange(config.Resources, config2.Resources) || nilChange(config.HPASpec, config2.HPASpec) || nilChange(config.Strategy, config2.Strategy) {
		return true
	}

	if config.Resources != nil {
		if nilChange(config.Resources.Requests, config2.Resources.Requests) || nilChange(config.Resources.Limits, config2.Resources.Limits) {
			return true
		}

		if config.Resources.Limits != nil {
			if !resourceClaimsEqual(config.Resources.Limits, config2.Resources.Limits) {
				return true
			}
		}

		if config.Resources.Requests != nil {
			if !resourceClaimsEqual(config.Resources.Requests, config2.Resources.Requests) {
				return true
			}
		}
	}

	if config.HPASpec != nil {
		if nilChange(config.HPASpec.MinReplicas, config2.HPASpec.MinReplicas) || nilChange(config.HPASpec.MaxReplicas, config2.HPASpec.MaxReplicas) {
			return true
		}

		if config.HPASpec.MinReplicas != nil && *config.HPASpec.MinReplicas != *config2.HPASpec.MinReplicas {
			return true
		}

		if config.HPASpec.MaxReplicas != nil && *config.HPASpec.MaxReplicas != *config2.HPASpec.MaxReplicas {
			return true
		}
	}

	if config.Strategy != nil {
		if nilChange(config.Strategy.RollingUpdate.MaxSurge, config2.Strategy.RollingUpdate.MaxSurge) || nilChange(config.Strategy.RollingUpdate.MaxUnavailable, config2.Strategy.RollingUpdate.MaxUnavailable) {
			return true
		}

		if *config.Strategy.RollingUpdate.MaxSurge != *config2.Strategy.RollingUpdate.MaxSurge {
			return true
		}

		if *config.Strategy.RollingUpdate.MaxUnavailable != *config2.Strategy.RollingUpdate.MaxUnavailable {
			return true
		}
	}

	return false
}

func resourceClaimsEqual(a, b *operatorv1alpha1.ResourceClaims) bool {
	if nilChange(a.Memory, b.Memory) || nilChange(a.Cpu, b.Cpu) {
		return false
	}

	if a.Memory != nil && *a.Memory != *b.Memory {
		return false
	}

	if a.Cpu != nil && *a.Cpu != *b.Cpu {
		return false
	}

	return true
}

func nilChange(a, b interface{}) bool {
	aNil := a == nil || reflect.ValueOf(a).IsNil()
	bNil := b == nil || reflect.ValueOf(b).IsNil()
	return (aNil) != (bNil)
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
