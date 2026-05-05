package restart_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// BenchmarkRestart_SingleDeployment measures the cost of restarting a single pod
// owned by a Deployment.
func BenchmarkRestart_SingleDeployment(b *testing.B) {
	ctx := context.Background()
	logger := logr.Discard()

	deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}}
	podList := v1.PodList{Items: []v1.Pod{podFixture("p1", "test-ns", "Deployment", "owner")}}

	c := benchFakeClient(b, deployment)
	r := restart.NewActionRestarter(c, &logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Restart(ctx, &podList, false)
	}
}

// BenchmarkRestart_ManyPodsOneDeployment measures deduplication: N pods all owned
// by the same Deployment should result in exactly one patch.
func BenchmarkRestart_ManyPodsOneDeployment(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			deployment := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "owner", Namespace: "test-ns"}}
			pods := make([]v1.Pod, n)
			for i := range pods {
				pods[i] = podFixture(fmt.Sprintf("p%d", i), "test-ns", "Deployment", "owner")
			}
			podList := v1.PodList{Items: pods}

			c := benchFakeClient(b, deployment)
			r := restart.NewActionRestarter(c, &logger)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = r.Restart(ctx, &podList, false)
			}
		})
	}
}

// BenchmarkRestart_ManyPodsDistinctDeployments measures the cost when every pod
// belongs to a different Deployment (no deduplication possible).
func BenchmarkRestart_ManyPodsDistinctDeployments(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			objects := make([]client.Object, n)
			pods := make([]v1.Pod, n)
			for i := range pods {
				name := fmt.Sprintf("owner%d", i)
				objects[i] = &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "test-ns"}}
				pods[i] = podFixture(fmt.Sprintf("p%d", i), "test-ns", "Deployment", name)
			}
			podList := v1.PodList{Items: pods}

			c := benchFakeClient(b, objects...)
			r := restart.NewActionRestarter(c, &logger)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = r.Restart(ctx, &podList, false)
			}
		})
	}
}

// BenchmarkRestart_MixedOwnerKinds measures a realistic workload with Deployments,
// DaemonSets, StatefulSets, and bare ReplicaSets all present simultaneously.
func BenchmarkRestart_MixedOwnerKinds(b *testing.B) {
	ctx := context.Background()
	logger := logr.Discard()

	objects := []client.Object{
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "dep", Namespace: "test-ns"}},
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "ds", Namespace: "test-ns"}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Name: "sts", Namespace: "test-ns"}},
		&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{Name: "rs", Namespace: "test-ns"}},
	}
	pods := []v1.Pod{
		podFixture("p1", "test-ns", "Deployment", "dep"),
		podFixture("p2", "test-ns", "DaemonSet", "ds"),
		podFixture("p3", "test-ns", "StatefulSet", "sts"),
		podFixture("p4", "test-ns", "ReplicaSet", "rs"),
	}
	podList := v1.PodList{Items: pods}

	c := benchFakeClient(b, objects...)
	r := restart.NewActionRestarter(c, &logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Restart(ctx, &podList, false)
	}
}

// BenchmarkRestart_WarningPaths measures the fast-path where pods produce warnings
// (no owner, Job owner) so no k8s mutations are issued.
func BenchmarkRestart_WarningPaths(b *testing.B) {
	ctx := context.Background()
	logger := logr.Discard()

	pods := make([]v1.Pod, 0, 100)
	for i := 0; i < 50; i++ {
		pods = append(pods, podWithoutOwnerFixture(fmt.Sprintf("no-owner-%d", i), "test-ns"))
		pods = append(pods, podFixture(fmt.Sprintf("job-pod-%d", i), "test-ns", "Job", fmt.Sprintf("job%d", i)))
	}
	podList := v1.PodList{Items: pods}

	c := benchFakeClient(b)
	r := restart.NewActionRestarter(c, &logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = r.Restart(ctx, &podList, false)
	}
}

// BenchmarkRestart_ReplicaSetChain measures the extra Get calls involved when a pod
// is owned by a ReplicaSet that is itself owned by a Deployment.
func BenchmarkRestart_ReplicaSetChain(b *testing.B) {
	ctx := context.Background()
	logger := logr.Discard()

	objects := []client.Object{
		&appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			Name: "rs", Namespace: "test-ns",
			OwnerReferences: []metav1.OwnerReference{{Name: "dep", Kind: "Deployment"}},
		}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "dep", Namespace: "test-ns"}},
	}
	podList := v1.PodList{Items: []v1.Pod{podFixture("p1", "test-ns", "ReplicaSet", "rs")}}

	c := benchFakeClient(b, objects...)
	r := restart.NewActionRestarter(c, &logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {

		_, _ = r.Restart(ctx, &podList, false)
	}
}

func benchFakeClient(b *testing.B, objects ...client.Object) client.Client {
	b.Helper()
	return fake.NewClientBuilder().WithScheme(buildScheme()).WithObjects(objects...).Build()
}
