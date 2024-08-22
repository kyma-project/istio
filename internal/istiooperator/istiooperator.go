package istiooperator

import (
	_ "embed"
	"errors"
	"fmt"

	"github.com/coreos/go-semver/semver"
	operatorv1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

//go:embed istio-operator.yaml
var ProductionOperator []byte

//go:embed istio-operator-light.yaml
var EvaluationOperator []byte

const (
	MergedIstioOperatorFile = "merged-istio-operator.yaml"
)

type IstioImageVersion struct {
	semanticVersion *semver.Version
}

func NewIstioImageVersionFromTag(tag string) (IstioImageVersion, error) {
	semVersion, err := semver.NewVersion(tag)
	if err != nil {
		return IstioImageVersion{}, err
	}
	return IstioImageVersion{semanticVersion: semVersion}, nil
}

func (i *IstioImageVersion) Version() string {
	return fmt.Sprintf("%d.%d.%d", i.semanticVersion.Major, i.semanticVersion.Minor, i.semanticVersion.Patch)
}

func (i *IstioImageVersion) Flavor() string {
	return string(i.semanticVersion.PreRelease)
}

func (i *IstioImageVersion) Tag() string {
	return i.semanticVersion.String()
}

func (i *IstioImageVersion) Empty() bool {
	return i.semanticVersion == nil
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

func (m *IstioMerger) GetIstioImageVersion() (IstioImageVersion, error) {
	// We can always use the Production cluster size here, because we have tests verifying that Production
	// and Evaluation have the same version.
	iop, err := m.GetIstioOperator(clusterconfig.Production)
	if err != nil {
		return IstioImageVersion{}, err
	}

	return NewIstioImageVersionFromTag(iop.Spec.Tag.GetStringValue())
}

func (m *IstioMerger) GetIstioOperator(clusterSize clusterconfig.ClusterSize) (iopv1alpha1.IstioOperator, error) {
	var istioOperator []byte
	switch clusterSize {
	case clusterconfig.Production:
		istioOperator = ProductionOperator
	case clusterconfig.Evaluation:
		istioOperator = EvaluationOperator
	default:
		return iopv1alpha1.IstioOperator{}, errors.New("unsupported cluster size")
	}
	toBeInstalledIop := iopv1alpha1.IstioOperator{}
	err := yaml.Unmarshal(istioOperator, &toBeInstalledIop)
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
