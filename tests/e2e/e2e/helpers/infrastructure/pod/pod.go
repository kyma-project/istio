package pod

import (
	"fmt"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

func CreatePod(t *testing.T, k8sClient client.Client, pod *v1.Pod) error {
	t.Helper()
	setup.DeclareCleanup(t, func() {
		t.Logf("Cleaning Pod: name: %s, namespace: %s", pod.Name, pod.Namespace)
		err := k8sClient.Delete(setup.GetCleanupContext(), pod)
		if err != nil {
			t.Logf("Failed to delete pod: name: %s, namespace: %s, because: %s", pod.Namespace, pod.Name, err)
		}
	})

	t.Logf("Creating Pod: name: %s, namespace: %s", pod.Name, pod.Namespace)
	err := k8sClient.Create(t.Context(), pod)
	if err != nil {
		return fmt.Errorf("failed to create pod: name: %s, namespace: %s, because: %w", pod.Namespace, pod.Name, err)
	}
	return nil
}

func GetPod(t *testing.T, k8sClient client.Client, namespaceName string, name string) (*v1.Pod, error) {
	t.Helper()
	t.Logf("Getting Pod: name: %s, namespace: %s", name, namespaceName)
	returnedPod := &v1.Pod{}
	err := k8sClient.Get(t.Context(), types.NamespacedName{Namespace: namespaceName, Name: name}, returnedPod)
	if err != nil {
		return nil, err
	}
	return returnedPod, nil
}
