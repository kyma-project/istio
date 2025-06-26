package unstructured

import (
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	unstructured2 "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func CreateObjectFromUnstructured(t *testing.T, k8sClient client.Client, unstructuredYaml *unstructured2.Unstructured) error {
	t.Logf("Creating object from unstructured: name: %s, namespace: %s, kind %s",
		unstructuredYaml.GetName(), unstructuredYaml.GetNamespace(), unstructuredYaml.GetKind())

	setup.DeclareCleanup(t, func() {
		deleteErr := k8sClient.Delete(setup.GetCleanupContext(), unstructuredYaml)
		if deleteErr != nil {
			t.Logf("Failed to delete object: %s", unstructuredYaml.GetName())
		}
	})

	if createErr := k8sClient.Create(t.Context(), unstructuredYaml); createErr != nil {
		return createErr
	}

	return nil
}

func GetObjectFromUnstructured(t *testing.T, k8sClient client.Client, unstructuredYaml *unstructured2.Unstructured) (*unstructured2.Unstructured, error) {
	t.Helper()
	t.Logf("Getting object from unstructured: name: %s, namespace: %s, kind %s",
		unstructuredYaml.GetName(), unstructuredYaml.GetNamespace(), unstructuredYaml.GetKind())

	retrievedObject := unstructured2.Unstructured{}
	retrievedObject.SetGroupVersionKind(unstructuredYaml.GetObjectKind().GroupVersionKind())
	namespacedName := getNamespacedName(unstructuredYaml.GetNamespace(), unstructuredYaml.GetName())
	getErr := k8sClient.Get(t.Context(), namespacedName, &retrievedObject)
	if getErr != nil {
		return nil, getErr
	}
	return &retrievedObject, nil
}

func getNamespacedName(namespaceName string, name string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespaceName,
		Name:      name,
	}
}
