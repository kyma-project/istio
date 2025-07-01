package yamlfile

import (
	unstructured2 "github.com/kyma-project/istio/operator/tests/e2e/e2e/helpers/infrastructure/unstructured"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func readYaml(filePath string) (*unstructured.Unstructured, error) {
	fileYaml, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	unstructuredObject := &unstructured.Unstructured{}
	if yamlErr := yaml.Unmarshal(fileYaml, unstructuredObject); yamlErr != nil {
		return nil, yamlErr
	}

	return unstructuredObject, nil
}

func CreateObjectFromYamlFile(t *testing.T, k8sClient client.Client, filePath string) error {
	t.Helper()
	t.Logf("Creating object from yaml file: %s", filePath)
	unstructuredYaml, err := readYaml(filePath)
	if err != nil {
		return err
	}

	return unstructured2.CreateObjectFromUnstructured(t, k8sClient, unstructuredYaml)
}

func GetObjectFromYamlFile(t *testing.T, k8sClient client.Client, filePath string) (*unstructured.Unstructured, error) {
	t.Helper()
	t.Logf("Getting object from yaml file: %s", filePath)
	unstructuredYaml, err := readYaml(filePath)
	if err != nil {
		return nil, err
	}

	return unstructured2.GetObjectFromUnstructured(t, k8sClient, unstructuredYaml)
}
