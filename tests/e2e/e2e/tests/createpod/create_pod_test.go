package createpod_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kyma-project/istio/operator/tests/e2e/e2e/executor"
	e2ePod "github.com/kyma-project/istio/operator/tests/e2e/e2e/steps/infrastructure/pod"
)

func TestPodCreation(t *testing.T) {
	t.Parallel()
	// Setup Infra

	t.Run("test", func(t *testing.T) {
		t.Parallel()
		testExecutor := executor.NewExecutor(t)
		defer testExecutor.Cleanup()

		createPod := &e2ePod.Create{
			Pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-pod",
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
			},
		}

		err := testExecutor.RunStep(createPod)
		require.NoError(t, err)

		podGetter := &e2ePod.Get{
			PodNamespace: "default",
			PodName:      "test-pod",
		}

		err = testExecutor.RunStep(podGetter)
		require.NoError(t, err)

		retrievedPod := podGetter.Output()
		assert.NotNil(t, retrievedPod)
	})
}
