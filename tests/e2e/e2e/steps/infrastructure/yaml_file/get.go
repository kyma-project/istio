package yaml_file

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"sync/atomic"
	"testing"
)

type Get struct {
	FilePath string

	Output atomic.Pointer[unstructured.Unstructured]
}

func (g *Get) Description() string {
	var _, current, _, _ = runtime.Caller(0)
	return fmt.Sprintf("%s: filePath=%s", current, g.FilePath)
}

func (g *Get) Execute(t *testing.T, ctx context.Context, k8sClient client.Client) error {
	unstructuredFromFile := unstructured.Unstructured{}
	yamlFile, err := os.ReadFile(g.FilePath)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(yamlFile, &unstructuredFromFile); err != nil {
		return err
	}

	unstructuredObject := unstructured.Unstructured{}
	unstructuredObject.SetGroupVersionKind(unstructuredFromFile.GetObjectKind().GroupVersionKind())
	if err := k8sClient.Get(ctx,
		types.NamespacedName{
			Namespace: unstructuredFromFile.GetNamespace(),
			Name:      unstructuredFromFile.GetName(),
		},
		&unstructuredObject,
	); err != nil {
		return err
	}

	g.Output.Store(&unstructuredObject)
	return nil
}

func (g *Get) Cleanup(context.Context, client.Client) error {
	return nil
}
