package pod

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"testing"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Get struct {
	Namespace string
	Name      string
	GVK       schema.GroupVersionKind

	Output *unstructured.Unstructured
}

func (p *Get) Description() string {
	return fmt.Sprintf("Get %v: name=%s, namespace=%s", p.GVK, p.Name, p.Namespace)
}

func (p *Get) Execute(t *testing.T, k8sClient client.Client) error {
	obj := &unstructured.Unstructured{}
	obj.SetGroupVersionKind(p.GVK)

	err := k8sClient.Get(t.Context(), types.NamespacedName{Namespace: p.Namespace, Name: p.Name}, obj)
	if err != nil {
		return err
	}

	p.Output = obj
	return nil
}
