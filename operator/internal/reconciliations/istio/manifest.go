package istio

import (
	"bytes"
	"errors"
	"os"
	"path"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
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

func merge(istioCR *operatorv1alpha1.Istio, istioOperatorFilePath string, workingDir string, data TemplateData) (string, error) {
	manifest, err := os.ReadFile(istioOperatorFilePath)
	if err != nil {
		return "", err
	}

	mergedManifest, err := applyIstioCR(istioCR, manifest)
	if err != nil {
		return "", err
	}

	templatedManifest, err := parseManifestWithTemplate(string(mergedManifest), data)
	if err != nil {
		return "", err
	}

	mergedIstioOperatorPath := path.Join(workingDir, mergedIstioOperatorFile)
	err = os.WriteFile(mergedIstioOperatorPath, templatedManifest, 0o644)
	if err != nil {
		return "", err
	}

	return mergedIstioOperatorPath, nil
}

func applyIstioCR(istioCR *operatorv1alpha1.Istio, operatorManifest []byte) ([]byte, error) {
	toBeInstalledIop := istioOperator.IstioOperator{}
	err := yaml.Unmarshal(operatorManifest, &toBeInstalledIop)
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
