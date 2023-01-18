package istio

import (
	"os"
	"path"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

var (
	mergedIstioOperatorFile = "merged-istio-operator.yaml"
)

func merge(istioCR *operatorv1alpha1.Istio, istioOperatorFilePath string, workingDir string) (string, error) {
	manifest, err := os.ReadFile(istioOperatorFilePath)
	if err != nil {
		return "", err
	}

	mergedManifest, err := applyIstioCR(istioCR, manifest)
	if err != nil {
		return "", err
	}

	mergedIstioOperatorPath := path.Join(workingDir, mergedIstioOperatorFile)
	err = os.WriteFile(mergedIstioOperatorPath, mergedManifest, 0o644)
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
