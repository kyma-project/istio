package manifest

import (
	"fmt"
	"os"

	operatorv1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"sigs.k8s.io/yaml"
)

var (
	DefaultIstioOperatorFile = "default-istio-operator-k3d.yaml"
)

func Merge(istioCR *operatorv1alpha1.Istio) (string, error) {
	b, err := os.ReadFile(DefaultIstioOperatorFile)
	if err != nil {
		return "", err
	}

	d, err := applyIstioCR(istioCR, b)
	if err != nil {
		return "", err
	}
	filePath := fmt.Sprintf("/tmp/%s", DefaultIstioOperatorFile)
	err = os.WriteFile(filePath, d, 0o644)
	if err != nil {
		return "", err
	}

	return filePath, nil
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
