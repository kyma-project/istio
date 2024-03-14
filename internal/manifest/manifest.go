package manifest

import (
	"os"
	"path"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

const (
	mergedIstioOperatorFile = "merged-istio-operator.yaml"
	workingDir              = "/tmp"
)

var readFileHandle = os.ReadFile

type Merger interface {
	Merge(baseManifestPath string, istioCR *operatorv1alpha2.Istio, overrides clusterconfig.ClusterConfiguration) (string, error)
	GetIstioOperator(baseManifestPath string) (iopv1alpha1.IstioOperator, error)
}

type IstioMerger struct {
	workingDir string
}

func NewDefaultIstioMerger() IstioMerger {
	return IstioMerger{
		workingDir: workingDir,
	}
}

func (m *IstioMerger) Merge(baseManifestPath string, istioCR *operatorv1alpha2.Istio, overrides clusterconfig.ClusterConfiguration) (string, error) {
	toBeInstalledIop, err := m.GetIstioOperator(baseManifestPath)
	if err != nil {
		return "", err
	}

	mergedManifest, err := applyIstioCR(istioCR, toBeInstalledIop)
	if err != nil {
		return "", err
	}

	manifestWithOverrides, err := clusterconfig.MergeOverrides(mergedManifest, overrides)
	if err != nil {
		return "", err
	}

	mergedIstioOperatorPath := path.Join(m.workingDir, mergedIstioOperatorFile)
	err = os.WriteFile(mergedIstioOperatorPath, manifestWithOverrides, 0o644)
	if err != nil {
		return "", err
	}

	return mergedIstioOperatorPath, nil
}

func (m *IstioMerger) GetIstioOperator(baseManifestPath string) (iopv1alpha1.IstioOperator, error) {
	manifest, err := readFileHandle(baseManifestPath)
	if err != nil {
		return iopv1alpha1.IstioOperator{}, err
	}

	toBeInstalledIop := iopv1alpha1.IstioOperator{}
	err = yaml.Unmarshal(manifest, &toBeInstalledIop)
	if err != nil {
		return iopv1alpha1.IstioOperator{}, err
	}
	return toBeInstalledIop, nil
}

func applyIstioCR(istioCR *operatorv1alpha2.Istio, toBeInstalledIop iopv1alpha1.IstioOperator) ([]byte, error) {
	_, err := istioCR.MergeInto(toBeInstalledIop)
	if err != nil {
		return nil, err
	}

	outputManifest, err := yaml.Marshal(toBeInstalledIop)
	if err != nil {
		return nil, err
	}

	return outputManifest, nil
}
