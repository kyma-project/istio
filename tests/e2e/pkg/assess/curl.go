package assess

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
	"sigs.k8s.io/e2e-framework/klient/wait/conditions"
	"sigs.k8s.io/e2e-framework/pkg/envconf"
	"sigs.k8s.io/e2e-framework/pkg/types"
)

const (
	podName       = "curl"
	containerName = "curl"
)

func RunCurlClusterStep(command string) types.StepFunc {
	return func(ctx context.Context, t *testing.T, cfg *envconf.Config) context.Context {
		cmd := strings.Split(command, " ")
		podID := envconf.RandomName(podName, 16)
		pod := corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: podID, Namespace: cfg.Namespace()},
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
		r, err := resources.New(cfg.Client().RESTConfig())
		require.NoError(t, err)

		t.Logf("Creating pod: %s", podID)
		err = r.Create(t.Context(), &pod)
		require.NoError(t, err)

		require.NoError(t, wait.For(conditions.New(r).PodRunning(&pod)))

		// this may cause confusion because it modifies input command provided by the user
		cmd = append(cmd, "--fail-with-body")

		var stdout, stderr bytes.Buffer
		assert.NoError(t, r.ExecInPod(t.Context(), pod.GetNamespace(), pod.GetName(), containerName, cmd, &stdout, &stderr))
		t.Logf("[%s] stdout: %v", podID, stdout.String())
		t.Logf("[%s] stderr: %v", podID, stderr.String())
		return ctx
	}
}
