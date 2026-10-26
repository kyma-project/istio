/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RestartAnnotation marks a pod template so the owning controller performs a
// rolling restart. Reused from the previous istio-operator restarter so pods
// restarted by either implementation share the same marker.
const RestartAnnotation = "istio-operator.kyma-project.io/restartedAt"

// FingerprintAnnotation records the config fingerprint that caused the last
// restart of a workload. Several evaluators vote Restart purely from the
// desired-vs-applied Istio config, which the restart action never changes, so
// without a record of "already restarted for this config" they would vote
// Restart on every reconcile and the workload would roll forever. The guard
// compares this stamp against the current fingerprint and restarts only when
// they differ, yielding exactly one restart per distinct config state.
const FingerprintAnnotation = "istio-operator.kyma-project.io/restartFingerprint"

// now is overridable for tests. It is a package-level variable and is NOT
// safe for concurrent mutation: tests that override it must run sequentially
// (do not call t.Parallel on tests that use withFixedNow).
var now = time.Now

// patchTemplateRestart stamps the restart annotation and the config fingerprint
// onto the pod template of a Deployment, StatefulSet, or DaemonSet via strategic
// merge patch. Both annotations live in the pod template so they propagate to
// every new pod produced by the rolling restart, and the guard that prevents
// duplicate restarts reads from the same location.
func patchTemplateRestart(ctx context.Context, c client.Client, obj client.Object, tmpl *corev1.PodTemplateSpec, fingerprint string) error {
	patch := client.StrategicMergeFrom(obj.DeepCopyObject().(client.Object))
	if tmpl.Annotations == nil {
		tmpl.Annotations = map[string]string{}
	}
	tmpl.Annotations[RestartAnnotation] = now().UTC().Format(time.RFC3339)
	tmpl.Annotations[FingerprintAnnotation] = fingerprint
	return c.Patch(ctx, obj, patch)
}

func restartDeployment(ctx context.Context, c client.Client, d *appsv1.Deployment, fingerprint string) error {
	return patchTemplateRestart(ctx, c, d, &d.Spec.Template, fingerprint)
}

func restartStatefulSet(ctx context.Context, c client.Client, s *appsv1.StatefulSet, fingerprint string) error {
	return patchTemplateRestart(ctx, c, s, &s.Spec.Template, fingerprint)
}

func restartDaemonSet(ctx context.Context, c client.Client, d *appsv1.DaemonSet, fingerprint string) error {
	return patchTemplateRestart(ctx, c, d, &d.Spec.Template, fingerprint)
}

// restartReplicaSetPods deletes the given pods so the ReplicaSet controller
// recreates them, then stamps both the restart timestamp and the config
// fingerprint into the ReplicaSet's pod template annotations. The guard that
// prevents duplicate restarts reads from the pod template, consistent with the
// other workload kinds. Delete is the only reliable trigger for ownerless
// ReplicaSets — patching the template alone does not force existing pods to be
// replaced because the pod-template hash stays identical.
func restartReplicaSetPods(ctx context.Context, c client.Client, rs *appsv1.ReplicaSet, pods []*corev1.Pod, fingerprint string) error {
	for _, p := range pods {
		if err := c.Delete(ctx, p); err != nil && client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("delete pod %s/%s: %w", p.Namespace, p.Name, err)
		}
	}
	return patchTemplateRestart(ctx, c, rs, &rs.Spec.Template, fingerprint)
}

// alreadyRestartedForConfig reports whether stamped equals the current config
// fingerprint, meaning the workload was already restarted for this config and
// must not be restarted again. An empty fingerprint never matches, so a missing
// or not-yet-known config can never suppress a restart by accident.
func alreadyRestartedForConfig(stamped, fingerprint string) bool {
	return fingerprint != "" && stamped == fingerprint
}
