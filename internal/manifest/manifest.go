package manifest

import (
	"bytes"
	"errors"
	"os"
	"path"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/internal/clusterconfig"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"

	"text/template"
)

const (
	mergedIstioOperatorFile = "merged-istio-operator.yaml"
	workingDir              = "/tmp"
)

type TemplateData struct {
	IstioVersion   string
	IstioImageBase string
	ModuleVersion  string
}

type Merger interface {
	Merge(baseManifestPath string, istioCR *operatorv1alpha1.Istio, templateData TemplateData, overrides clusterconfig.ClusterConfiguration) (string, error)
	GetIstioOperator(baseManifestPath string) (istioOperator.IstioOperator, error)
}

type IstioMerger struct {
	workingDir string
}

func NewDefaultIstioMerger() IstioMerger {
	return IstioMerger{
		workingDir: workingDir,
	}
}

func (m *IstioMerger) Merge(baseManifestPath string, istioCR *operatorv1alpha1.Istio, templateData TemplateData, overrides clusterconfig.ClusterConfiguration) (string, error) {
	toBeInstalledIop, err := m.GetIstioOperator(baseManifestPath)
	if err != nil {
		return "", err
	}
	mergedManifest, err := applyIstioCR(istioCR, toBeInstalledIop)
	if err != nil {
		return "", err
	}

	templatedManifest, err := parseManifestWithTemplate(string(mergedManifest), templateData)
	if err != nil {
		return "", err
	}

	manifestWithOverrides, err := clusterconfig.MergeOverrides(templatedManifest, overrides)
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

func (m *IstioMerger) GetIstioOperator(baseManifestPath string) (istioOperator.IstioOperator, error) {
	manifest, err := os.ReadFile(baseManifestPath)
	if err != nil {
		return istioOperator.IstioOperator{}, err
	}

	toBeInstalledIop := istioOperator.IstioOperator{}
	err = yaml.Unmarshal(manifest, &toBeInstalledIop)
	if err != nil {
		return istioOperator.IstioOperator{}, err
	}
	return toBeInstalledIop, nil
}

func applyIstioCR(istioCR *operatorv1alpha1.Istio, toBeInstalledIop istioOperator.IstioOperator) ([]byte, error) {

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

func parseManifestWithTemplate(templateRaw string, data TemplateData) ([]byte, error) {
	if data.IstioVersion == "" {
		return nil, errors.New("IstioVersion cannot be empty")
	}

	if data.IstioImageBase == "" {
		return nil, errors.New("IstioImageBase cannot be empty")
	}

	tmpl, err := template.New("tmpl").Parse(templateRaw)
	if err != nil {
		return nil, err
	}

	var resource bytes.Buffer
	err = tmpl.Execute(&resource, data)
	if err != nil {
		return nil, err
	}
	return resource.Bytes(), nil
}
