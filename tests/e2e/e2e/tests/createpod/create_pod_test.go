package createpod_test

import (
	"fmt"
	podHelper "github.com/kyma-project/istio/operator/tests/e2e/e2e/helpers/infrastructure/pod"
	"github.com/kyma-project/istio/operator/tests/e2e/e2e/setup"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodCreation(t *testing.T) {
	k8sClient := setup.ClientFromKubeconfig(t)
	testId := setup.GenerateRandomTestId()
	namespaceName := fmt.Sprintf("test-ns-%s", testId)
	setup.CreateNamespaceForTest(t, k8sClient, namespaceName)

	t.Run("test", func(t *testing.T) {
		t.Parallel()

		// given
		podName := "pod-test"
		pod := createNginxPodStruct(namespaceName, podName)

		// when
		err := podHelper.CreatePod(t, k8sClient, pod)

		// then
		require.NoError(t, err)

		retrievedPod, err := podHelper.GetPod(t, k8sClient, namespaceName, podName)
		require.NoError(t, err)
		assert.NotNil(t, retrievedPod)
	})
}

func createNginxPodStruct(namespace string, name string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "test-container",
					Image: "nginx:latest",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
							Name:          "http",
						},
					},
				},
			},
		},
	}
}
