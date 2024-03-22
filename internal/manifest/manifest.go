package manifest

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/coreos/go-semver/semver"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	"github.com/kyma-project/istio/operator/internal/istiooperator"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

const (
	MergedIstioOperatorFile = "merged-istio-operator.yaml"
)

type IstioImageVersion struct {
	Version string
	Flavor  string
}

func NewIstioImageVersionFromTag(tag string) (IstioImageVersion, error) {
	semVersion, err := semver.NewVersion(tag)
	if err != nil {
		return IstioImageVersion{}, err
	}
	return IstioImageVersion{
		Version: fmt.Sprintf("%d.%d.%d", semVersion.Major, semVersion.Minor, semVersion.Patch),
		Flavor:  string(semVersion.PreRelease),
	}, nil
}

func (i *IstioImageVersion) Tag() string {
	return fmt.Sprintf("%s-%s", i.Version, i.Flavor)
}

type Merger interface {
	Merge(clusterSize clusterconfig.ClusterSize, istioCR *operatorv1alpha2.Istio, overrides clusterconfig.ClusterConfiguration) (string, error)
	GetIstioOperator(clusterSize clusterconfig.ClusterSize) (iopv1alpha1.IstioOperator, error)
	GetIstioImageVersion() (IstioImageVersion, error)
}

type IstioMerger struct {
	workingDir string
}

func NewDefaultIstioMerger() IstioMerger {
	return IstioMerger{
		workingDir: "/tmp",
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
	mergedIstioOperatorPath := path.Join(m.workingDir, MergedIstioOperatorFile)
	err = os.WriteFile(mergedIstioOperatorPath, manifestWithOverrides, 0o644)
	if err != nil {
		return "", err
	}
	return mergedIstioOperatorPath, nil
}

func (m *IstioMerger) GetIstioImageVersion() (IstioImageVersion, error) {
	iop, err := m.GetIstioOperator(clusterconfig.Production)
	if err != nil {
		return IstioImageVersion{}, err
	}

	return NewIstioImageVersionFromTag(iop.Spec.Tag.GetStringValue())
}

func (m *IstioMerger) GetIstioOperator(clusterSize clusterconfig.ClusterSize) (iopv1alpha1.IstioOperator, error) {
	var manifest []byte
	switch clusterSize {
	case clusterconfig.Production:
		manifest = istiooperator.ProductionOperator
	case clusterconfig.Evaluation:
		manifest = istiooperator.EvaluationOperator
	default:
		return iopv1alpha1.IstioOperator{}, errors.New("unsupported cluster size")
	}
	toBeInstalledIop := iopv1alpha1.IstioOperator{}
	err := yaml.Unmarshal(manifest, &toBeInstalledIop)
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
