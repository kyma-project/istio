package yaml_file

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"os"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
	"testing"
)

type Create struct {
	FilePath string
}

func (c *Create) Description() string {
	var _, current, _, _ = runtime.Caller(0)
	return fmt.Sprintf("%s: filePath=%s", current, c.FilePath)
}

func (c *Create) Execute(t *testing.T, ctx context.Context, k8sClient client.Client) error {
	unstructuredObject := &unstructured.Unstructured{}
	fileYaml, err := os.ReadFile(c.FilePath)
	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(fileYaml, unstructuredObject); err != nil {
		return err
	}

	executor.Debugf(t, "Creating object from YAML:\n%+v", string(fileYaml))
	if err := k8sClient.Create(ctx, unstructuredObject); err != nil {
		return err
	}

	return nil
}

func (c *Create) Cleanup(ctx context.Context, k8sClient client.Client) error {
	unstructuredObject := &unstructured.Unstructured{}
	fileYaml, err := os.ReadFile(c.FilePath)
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
