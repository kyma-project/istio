package namespace

import (
	"testing"

	v1 "k8s.io/api/core/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

func LabelNamespaceWithIstioInjection(t *testing.T, namespace string) error {
	t.Helper()
	t.Logf("Labeling namespace %s with istio injection label", namespace)

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
	}

	ns := &v1.Namespace{}
	err = r.Get(t.Context(), "default", "", ns)
	if err != nil {
		t.Logf("Failed to get default namespace: %v", err)
		return err
	}

	r.Label(ns, map[string]string{
		"istio-injection": "enabled",
	})

	err = r.Update(t.Context(), ns)
	if err != nil {
		t.Logf("Failed to update namespace: %v", err)
		return err
	}

	return nil
}

func LabelNamespaceWithAmbient(t *testing.T, namespace string) error {
	t.Helper()
	t.Logf("Labeling namespace %s with ambient dataplane mode label", namespace)

	r, err := client.ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	ns := &v1.Namespace{}
	err = r.Get(t.Context(), namespace, "", ns)
	if err != nil {
		t.Logf("Failed to get namespace %s: %v", namespace, err)
		return err
	}

	r.Label(ns, map[string]string{
		"istio.io/dataplane-mode": "ambient",
	})

	err = r.Update(t.Context(), ns)
	if err != nil {
		t.Logf("Failed to update namespace: %v", err)
		return err
	}

	return nil
}
