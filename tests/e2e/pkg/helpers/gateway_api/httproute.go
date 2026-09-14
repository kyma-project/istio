package gateway_api

import (
	"testing"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
)

// CreateHTTPRoute creates a minimal Gateway API HTTPRoute and registers cleanup.
func CreateHTTPRoute(t *testing.T, name, namespace string) (*unstructured.Unstructured, error) {
	t.Helper()
	t.Logf("creating HTTPRoute %s/%s", namespace, name)

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return nil, err
	}

	httpRoute := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "HTTPRoute",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"rules": []interface{}{
					map[string]interface{}{
						"matches": []interface{}{
							map[string]interface{}{
								"path": map[string]interface{}{
									"type":  "PathPrefix",
									"value": "/",
								},
							},
						},
					},
				},
			},
		},
	}

	err = r.Create(t.Context(), httpRoute)
	if err != nil {
		if !k8serrors.IsAlreadyExists(err) {
			t.Logf("Failed to create HTTPRoute %s/%s: %v", namespace, name, err)
			return nil, err
		}
		t.Logf("HTTPRoute %s/%s already exists", namespace, name)
	} else {
		t.Logf("HTTPRoute %s/%s created", namespace, name)
	}

	setup.DeclareCleanup(t, func() {
		toDelete := &unstructured.Unstructured{}
		toDelete.SetAPIVersion("gateway.networking.k8s.io/v1")
		toDelete.SetKind("HTTPRoute")
		toDelete.SetName(name)
		toDelete.SetNamespace(namespace)
		if delErr := r.Delete(setup.GetCleanupContext(), toDelete); delErr != nil && !k8serrors.IsNotFound(delErr) {
			t.Logf("Failed to delete HTTPRoute %s/%s: %v", namespace, name, delErr)
		}
	})

	return httpRoute, nil
}
