//go:build experimental

package istiooperator

import (
	"github.com/imdario/mergo"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	istiov1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"k8s.io/apimachinery/pkg/util/errors"
	"os"
	"path"
)

func (m *IstioMerger) Merge(clusterSize clusterconfig.ClusterSize, istioCR *operatorv1alpha2.Istio, overrides clusterconfig.ClusterConfiguration) (string, error) {
	toBeInstalledIop, err := m.GetIstioOperator(clusterSize)
	if err != nil {
		return "", err
	}

	if err := ParseExperimentalFeatures(istioCR, &toBeInstalledIop); err != nil {
		return "", err
	}
	mergedManifest, err := applyIstioCR(istioCR, toBeInstalledIop)
	if err != nil {
		return "", err
	}
	iopWithOverrides, err := clusterconfig.MergeOverrides(mergedManifest, overrides)
	if err != nil {
		return "", err
	}
	mergedIstioOperatorPath := path.Join(m.workingDir, MergedIstioOperatorFile)
	err = os.WriteFile(mergedIstioOperatorPath, iopWithOverrides, 0o644)
	if err != nil {
		return "", err
	}
	return mergedIstioOperatorPath, nil
}

// ParseExperimentalFeatures parses experimental options defined in Istio CR
// and sets the required features in the output operator CR.
// Handles changes in ExperimentalFeaturesApplied condition which is only managed
// in experimental flavour of image
func ParseExperimentalFeatures(istioCR *operatorv1alpha2.Istio, op *istiov1alpha1.IstioOperator) error {
	if istioCR.Spec.Experimental == nil {
		return nil
	}
	var errs []error
	if istioCR.Spec.Experimental.EnableAlphaGatewayAPI {
		err := enableGatewayAlphaAPI(op)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if istioCR.Spec.Experimental.EnableMultiNetworkDiscoverGatewayAPI {
		err := enableMultiNetworkDiscoverGatewayAPI(op)
		if err != nil {
			errs = append(errs, err)
		}
	}
	// set condition based on errors collected
	if len(errs) > 0 {
		// return aggregation of all errors collected from parsed options
		return errors.NewAggregate(errs)
	}
	return nil
}
func enableGatewayAlphaAPI(op *istiov1alpha1.IstioOperator) error {
	env := v1alpha1.EnvVar{
		Name:  "PILOT_ENABLE_ALPHA_GATEWAY_API",
		Value: "true",
	}

	toMerge := istiov1alpha1.IstioOperator{Spec: &v1alpha1.IstioOperatorSpec{
		Components: &v1alpha1.IstioComponentSetSpec{
			Pilot: &v1alpha1.ComponentSpec{
				K8S: &v1alpha1.KubernetesResourcesSpec{
					Env: []*v1alpha1.EnvVar{&env}}},
		}}}

	return mergo.Merge(op, toMerge, mergo.WithAppendSlice)
}
func enableMultiNetworkDiscoverGatewayAPI(op *istiov1alpha1.IstioOperator) error {
	env := v1alpha1.EnvVar{
		Name:  "PILOT_MULTI_NETWORK_DISCOVER_GATEWAY_API",
		Value: "true",
	}

	toMerge := istiov1alpha1.IstioOperator{Spec: &v1alpha1.IstioOperatorSpec{
		Components: &v1alpha1.IstioComponentSetSpec{
			Pilot: &v1alpha1.ComponentSpec{
				K8S: &v1alpha1.KubernetesResourcesSpec{
					Env: []*v1alpha1.EnvVar{&env}}},
		}}}

	return mergo.Merge(op, toMerge, mergo.WithAppendSlice)
}
