package restart

import (
	"context"
	"fmt"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const RestartAnnotation = "istio-operator.kyma-project.io/restartedAt"

const HashAnnotation = "istio-operator.kyma-project.io/restartHash"

var Now = time.Now

func SetNowForTest(t *testing.T, fn func() time.Time) {
	t.Helper()
	prev := Now
	Now = fn
	t.Cleanup(func() { Now = prev })
}

func patchTemplateRestart(ctx context.Context, c client.Client, obj client.Object, tmpl *corev1.PodTemplateSpec, fingerprint string) error {
	patch := client.StrategicMergeFrom(obj.DeepCopyObject().(client.Object))
	if tmpl.Annotations == nil {
		tmpl.Annotations = map[string]string{}
	}
	tmpl.Annotations[RestartAnnotation] = Now().UTC().Format(time.RFC3339)
	tmpl.Annotations[HashAnnotation] = fingerprint
	return c.Patch(ctx, obj, patch)
}

// Deployment performs a rolling restart of d by stamping its pod template.
func Deployment(ctx context.Context, c client.Client, d *appsv1.Deployment, fingerprint string) error {
	return patchTemplateRestart(ctx, c, d, &d.Spec.Template, fingerprint)
}

// StatefulSet performs a rolling restart of s by stamping its pod template.
func StatefulSet(ctx context.Context, c client.Client, s *appsv1.StatefulSet, fingerprint string) error {
	return patchTemplateRestart(ctx, c, s, &s.Spec.Template, fingerprint)
}

// DaemonSet performs a rolling restart of d by stamping its pod template.
func DaemonSet(ctx context.Context, c client.Client, d *appsv1.DaemonSet, fingerprint string) error {
	return patchTemplateRestart(ctx, c, d, &d.Spec.Template, fingerprint)
}

// ReplicaSetPods deletes the given pods so the ReplicaSet controller recreates
// Delete is the only reliable trigger for ownerless ReplicaSets —
// patching the template alone does not force existing pods.
func ReplicaSetPods(ctx context.Context, c client.Client, rs *appsv1.ReplicaSet, pods []*corev1.Pod, fingerprint string) error {
	for _, p := range pods {
		if err := c.Delete(ctx, p); err != nil && client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("delete pod %s/%s: %w", p.Namespace, p.Name, err)
		}
	}
	return patchTemplateRestart(ctx, c, rs, &rs.Spec.Template, fingerprint)
}

func AlreadyRestartedForConfig(stamped, hash string) bool {
	return hash != "" && stamped == hash
}
