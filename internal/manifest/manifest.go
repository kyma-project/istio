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

var (
	mergedIstioOperatorFile = "merged-istio-operator.yaml"
)

type TemplateData struct {
	IstioVersion   string
	IstioImageBase string
}

type IstioMerger interface {
	Merge() (string, error)
}

type DefaultIstioMerger struct {
	istioCR               *operatorv1alpha1.Istio
	istioOperatorFilePath string
	workingDir            string
	data                  TemplateData
	overrides             clusterconfig.ClusterConfiguration
}

func NewDefaultIstioMerger(istioCR *operatorv1alpha1.Istio, istioOperatorFilePath string, workingDir string, templateData TemplateData, overrides clusterconfig.ClusterConfiguration) DefaultIstioMerger {
	return DefaultIstioMerger{
		istioCR:               istioCR,
		istioOperatorFilePath: istioOperatorFilePath,
		workingDir:            workingDir,
		data:                  templateData,
		overrides:             overrides,
	}
}

func (m DefaultIstioMerger) Merge() (string, error) {
	mergedManifest, err := createOperatorManifest(m.istioCR, m.istioOperatorFilePath)
	if err != nil {
		return "", err
	}

	templatedManifest, err := parseManifestWithTemplate(string(mergedManifest), m.data)
	if err != nil {
		return "", err
	}

	manifestWithOverrides, err := clusterconfig.MergeOverrides(templatedManifest, m.overrides)
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

func createOperatorManifest(istioCR *operatorv1alpha1.Istio, istioOperatorManifestPath string) ([]byte, error) {
	toBeInstalledIop, err := GetIstioOperator(istioOperatorManifestPath)
	if err != nil {
		return nil, err
	}

	_, err = istioCR.MergeInto(toBeInstalledIop)
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
		return nil, errors.New("IstioImageBase cannot be empty")
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

func GetIstioOperator(istioOperatorManifestPath string) (istioOperator.IstioOperator, error) {
	manifest, err := os.ReadFile(istioOperatorManifestPath)
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
