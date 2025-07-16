package infrastructure

import (
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceOptions struct {
	Labels              map[string]string
	IgnoreAlreadyExists bool
}

func CreateNamespace(t *testing.T, name string, options ...NamespaceOptions) error {
	t.Helper()
	r, err := ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
	if len(options) > 0 {
		opts := options[0]
		if opts.Labels != nil {
			ns.ObjectMeta.Labels = opts.Labels
		}
	}

	t.Log("Creating namespace: ", name)

	err = r.Create(t.Context(), ns)
	if err != nil {
		if len(options) > 0 && options[0].IgnoreAlreadyExists && k8serrors.IsAlreadyExists(err) {
			t.Logf("Namespace %s already exists, ignoring error as per options", name)
			return nil
		}
		return err
	}

	setup.DeclareCleanup(t, func() {
		t.Log("Deleting namespace: ", name)
		err := DeleteNamespace(t, name)
		if err != nil {
			t.Logf("Failed to delete namespace %s: %v", name, err)
		}
	})
	return nil
}

func DeleteNamespace(t *testing.T, name string) error {
	t.Helper()
	r, err := ResourcesClient(t)
	if err != nil {
		t.Logf("Failed to get resources client: %v", err)
		return err
	}
	return r.Delete(setup.GetCleanupContext(), &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}})
}
