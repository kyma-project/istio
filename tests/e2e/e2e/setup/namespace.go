package setup

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateNamespaceForTest(t *testing.T, k8sClient client.Client, name string) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf(name),
		},
	}

	DeclareCleanup(t, func() {
		t.Logf("Cleaning namespace %s", name)
		err := k8sClient.Delete(GetCleanupContext(), namespace)
		if err != nil {
			t.Logf("failed to delete namespace %s, err: %v", name, err)
		}
	})

	t.Logf("Creating Namespace: %+v", *namespace)

	err := k8sClient.Create(t.Context(), namespace)
	if err != nil {
		t.Fatal(err)
	}
}
