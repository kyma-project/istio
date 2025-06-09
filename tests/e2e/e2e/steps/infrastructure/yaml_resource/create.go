package yaml_resource

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"log"
	"os"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

type Create struct {
	FilePath string
}

func (c *Create) Description() string {
	var _, current, _, _ = runtime.Caller(1)
	return fmt.Sprintf("%s: filePath=%s", current, c.FilePath)
}

func (c *Create) Execute(ctx context.Context, k8sClient client.Client, debugLog *log.Logger) error {
	unstructuredObject := &unstructured.Unstructured{}
	fileYaml, err := os.ReadFile(c.FilePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(fileYaml, unstructuredObject); err != nil {
		return err
	}

	debugLog.Printf("Creating object from YAML: %+v", unstructuredObject.Object)
	if err := k8sClient.Create(ctx, unstructuredObject); err != nil {
		return err
	}

	return nil
}

func (p *Create) Cleanup(ctx context.Context, k8sClient client.Client) error {
	unstructuredObject := &unstructured.Unstructured{}
	fileYaml, err := os.ReadFile(p.FilePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(fileYaml, unstructuredObject); err != nil {
		return err
	}

	if err := k8sClient.Delete(ctx, unstructuredObject); err != nil {
		return err
	}

	return nil
}
