package curl

import (
	"bytes"
	infrahelpers "github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/infrastructure"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/setup"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
)

const (
	podName       = "curl"
	containerName = "curl"
)

func RunCurlCmdInCluster(t *testing.T, namespace string, cmd []string) error {
	t.Helper()

	r := infrahelpers.ResourcesClient(t)
	podID := envconf.RandomName(podName, 16)
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podID, Namespace: namespace},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Args:  []string{"sleep", "infinity"},
					Image: "curlimages/curl:8.14.1",
					Name:  containerName,
				},
			},
		},
	}

	t.Logf("Creating curl pod: %s", podID)
	err := r.Create(t.Context(), &pod)
	if err != nil {
		return err
	}
	setup.DeclareCleanup(t, func() {
		t.Log("Deleting pod: ", pod.Name)
		require.NoError(t, r.Delete(setup.GetCleanupContext(), &pod))
	})
	err = wait.For(conditions.New(r).PodRunning(&pod))
	if err != nil {
		return err
	}
	// this may cause confusion because it modifies input command provided by the user
	cmd = append(cmd, "--fail-with-body")

	var stdout, stderr bytes.Buffer
	err = r.ExecInPod(t.Context(), pod.GetNamespace(), pod.GetName(), containerName, cmd, &stdout, &stderr)
	t.Logf("[%s] stdout: %v", podID, stdout.String())
	t.Logf("[%s] stderr: %v", podID, stderr.String())
	if err != nil {
		return err
	}
	return nil
}
