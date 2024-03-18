package manifest

import (
	"errors"
	"os"
	"path"

	_ "embed"

	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

const (
	mergedIstioOperatorFile = "merged-istio-operator.yaml"
	workingDir              = "/tmp"
)

//go:embed istio-operator.yaml
var productionOperator []byte

//go:embed istio-operator-light.yaml
var evaluationOperator []byte

type Merger interface {
	Merge(clusterSize clusterconfig.ClusterSize, istioCR *operatorv1alpha2.Istio, overrides clusterconfig.ClusterConfiguration) (string, error)
	GetIstioOperator(clusterSize clusterconfig.ClusterSize) (iopv1alpha1.IstioOperator, error)
}

type ManifestGetter interface {
	GetBytes(clusterSize clusterconfig.ClusterSize) ([]byte, error)
}

type ManifestGetterImpl struct {
}

func (m *ManifestGetterImpl) GetBytes(clusterSize clusterconfig.ClusterSize) ([]byte, error) {
	switch clusterSize {
	case clusterconfig.Production:
		return productionOperator, nil
	case clusterconfig.Evaluation:
		return evaluationOperator, nil
	default:
		return nil, errors.New("unsupported cluster size")
	}
}

type IstioMerger struct {
	workingDir     string
	manifestGetter ManifestGetter
}

func NewDefaultIstioMerger() IstioMerger {
	return IstioMerger{
		workingDir:     workingDir,
		manifestGetter: &ManifestGetterImpl{},
	}
}

func (m *IstioMerger) Merge(clusterSize clusterconfig.ClusterSize, istioCR *operatorv1alpha2.Istio, overrides clusterconfig.ClusterConfiguration) (string, error) {
	toBeInstalledIop, err := m.GetIstioOperator(clusterSize)
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

func (m *IstioMerger) GetIstioOperator(clusterSize clusterconfig.ClusterSize) (iopv1alpha1.IstioOperator, error) {
	manifest, err := m.manifestGetter.GetBytes(clusterSize)
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
