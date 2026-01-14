package log

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
)

func StructToPrettyJson(t *testing.T, v interface{}) string {
	t.Helper()
	str, err := json.MarshalIndent(v, "", "    ")
	assert.NoError(t, err)
	return string(str)
}

func GetLogsFromIstioProxy(t *testing.T, podName, podNamespace string) ([]byte, error) {
	t.Helper()
	config := client.GetKubeConfig(t)
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Logf("Failed to create k8s client: %v", err)
		return nil, err
	}

	req := k8sClient.CoreV1().Pods(podNamespace).GetLogs(podName, &v1.PodLogOptions{
		Container: "istio-proxy",
	})

	logs, err := req.DoRaw(t.Context())
	if err != nil {
		t.Logf("Failed to get logs from istio-proxy container: %v", err)
		return nil, err
	}

	return logs, nil
}

func GetLogsFromPodContainer(t *testing.T, podName, podNamespace, containerName string) ([]byte, error) {
	t.Helper()
	config := client.GetKubeConfig(t)
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Logf("Failed to create k8s client: %v", err)
		return nil, err
	}

	req := k8sClient.CoreV1().Pods(podNamespace).GetLogs(podName, &v1.PodLogOptions{
		Container: containerName,
	})

	logs, err := req.DoRaw(t.Context())
	if err != nil {
		t.Logf("Failed to get logs from container %s of Pod %s: %v", containerName, podName, err)
		return nil, err
	}

	return logs, nil
}

// AssertContainerLogContains asserts that container logs contain a specific string
// It retrieves pods for the given deployment and checks logs with retry logic
func AssertContainerLogContains(t *testing.T, c *resources.Resources, deploymentName, namespace, containerName, expectedLog string) {
	t.Helper()

	podList := &corev1.PodList{}
	err := c.List(t.Context(), podList, resources.WithLabelSelector(fmt.Sprintf("app=%s", deploymentName)))
	require.NoError(t, err)
	require.NotEmpty(t, podList.Items, "No pods found for deployment %s", deploymentName)

	pod := podList.Items[0]

	err = wait.For(func(ctx context.Context) (done bool, err error) {
		logs, err := GetLogsFromPodContainer(t, pod.Name, namespace, containerName)
		if err != nil {
			t.Logf("Failed to get logs, retrying: %v", err)
			return false, nil // Retry on error
		}

		if len(logs) == 0 {
			t.Logf("No logs found yet, retrying...")
			return false, nil
		}

		logsStr := string(logs)
		if !strings.Contains(logsStr, expectedLog) {
			t.Logf("Expected log not found yet, retrying...")
			return false, nil
		}

		return true, nil
	}, wait.WithTimeout(30*time.Second), wait.WithInterval(2*time.Second))

	require.NoError(t, err, "Expected log '%s' not found in container %s logs", expectedLog, containerName)
}
