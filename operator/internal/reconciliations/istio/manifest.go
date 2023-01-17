package istio

import (
	"fmt"
	"os"
	"path"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

var (
	defaultManifestDir       = "manifests"
	defaultIstioOperatorFile = "default-istio-operator-k3d.yaml"
)

func merge(istioCR *operatorv1alpha1.Istio) (string, error) {
	istioOperatorFilePath := path.Join(defaultManifestDir, defaultIstioOperatorFile)
	manifest, err := os.ReadFile(istioOperatorFilePath)
	if err != nil {
		return "", err
	}

	mergedManifest, err := applyIstioCR(istioCR, manifest)
	if err != nil {
		return "", err
	}
	mergedManifestFilePath := fmt.Sprintf("/tmp/%s", defaultIstioOperatorFile)
	err = os.WriteFile(mergedManifestFilePath, mergedManifest, 0o644)
	if err != nil {
		return "", err
	}

	return mergedManifestFilePath, nil
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
