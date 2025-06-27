package yamlfile

import (
	"fmt"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type Get struct {
	FilePath string

	Output unstructured.Unstructured
}

func (g *Get) Description() string {
	return fmt.Sprintf("%s: filePath=%s", "Get resource based on file", g.FilePath)
}

func (g *Get) Execute(t *testing.T, k8sClient client.Client) error {
	unstructuredFromFile := unstructured.Unstructured{}
	yamlFile, err := os.ReadFile(g.FilePath)
	if err != nil {
		return err
	}
	if unmarshalErr := yaml.Unmarshal(yamlFile, &unstructuredFromFile); unmarshalErr != nil {
		return unmarshalErr
	}

	unstructuredObject := unstructured.Unstructured{}
	unstructuredObject.SetGroupVersionKind(unstructuredFromFile.GetObjectKind().GroupVersionKind())
	if getErr := k8sClient.Get(t.Context(),
		types.NamespacedName{
			Namespace: unstructuredFromFile.GetNamespace(),
			Name:      unstructuredFromFile.GetName(),
		},
		&unstructuredObject,
	); getErr != nil {
		return getErr
	}
	g.Output = unstructuredObject

	return nil
}
