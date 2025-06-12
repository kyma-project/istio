package yamlfile

import (
	"fmt"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
)

type Create struct {
	FilePath string
}

func (c *Create) Description() string {
	return fmt.Sprintf("%s: filePath=%s", "Create resource from file", c.FilePath)
}

func (c *Create) Execute(t *testing.T, k8sClient client.Client) error {
	unstructuredObject := &unstructured.Unstructured{}
	fileYaml, err := os.ReadFile(c.FilePath)
	if err != nil {
		return err
	}

	if yamlErr := yaml.Unmarshal(fileYaml, unstructuredObject); yamlErr != nil {
		return yamlErr
	}

	executor.Debugf(t, "Creating object from YAML:\n%+v", string(fileYaml))
	if createErr := k8sClient.Create(t.Context(), unstructuredObject); createErr != nil {
		return createErr
	}

	return nil
}

func (c *Create) Cleanup(t *testing.T, k8sClient client.Client) error {
	unstructuredObject := &unstructured.Unstructured{}
	fileYaml, err := os.ReadFile(c.FilePath)
	if err != nil {
		return err
	}

	if unmarshalErr := yaml.Unmarshal(fileYaml, unstructuredObject); unmarshalErr != nil {
		return unmarshalErr
	}

	if deleteErr := k8sClient.Delete(t.Context(), unstructuredObject); deleteErr != nil {
		return deleteErr
	}

	return nil
}
