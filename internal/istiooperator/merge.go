//go:build !experimental

package istiooperator

import (
	"os"
	"path"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/images"
)

func (m *IstioMerger) Merge(clusterSize clusterconfig.ClusterSize, istioCR *operatorv1alpha2.Istio, overrides clusterconfig.ClusterConfiguration, istioImagesHubTag images.HubTag) (string, error) {
	toBeInstalledIop, err := m.GetIstioOperator(clusterSize)
	if err != nil {
		return "", err
	}
	mergedManifest, err := applyIstioCR(istioCR, toBeInstalledIop)
	if err != nil {
		return "", err
	}
	manifestWithOverrideImagesHub, err := images.MergeHubTagConfiguration(mergedManifest, istioImagesHubTag)
	if err != nil {
		return "", err
	}
	manifestWithOverridePullSecret, err := images.MergePullSecretEnv(manifestWithOverrideImagesHub)
	if err != nil {
		return "", err
	}
	iopWithOverrides, err := clusterconfig.MergeOverrides(manifestWithOverridePullSecret, overrides)
	if err != nil {
		return "", err
	}
	mergedIstioOperatorPath := path.Join(m.workingDir, MergedIstioOperatorFile)
	err = os.WriteFile(mergedIstioOperatorPath, iopWithOverrides, 0o600)
	if err != nil {
		return "", err
	}
	return mergedIstioOperatorPath, nil
}
