package pods_test

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"sync"
	"testing"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/images"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/test/helpers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	benchScheme     *runtime.Scheme
	benchSchemeOnce sync.Once
)

func buildBenchScheme() *runtime.Scheme {
	benchSchemeOnce.Do(func() {
		benchScheme = runtime.NewScheme()
		_ = v1.AddToScheme(benchScheme)
		_ = clientgoscheme.AddToScheme(benchScheme)
	})
	return benchScheme
}

var benchExpectedImage = images.Image{Registry: "istio", Name: "proxyv2", Tag: "1.10.0"}

func benchClient(b *testing.B, objs ...client.Object) client.Client {
	b.Helper()
	stdlog.SetOutput(io.Discard) // silence ImageResourcesPredicate's log.Printf
	return fake.NewClientBuilder().
		WithScheme(buildBenchScheme()).
		WithIndex(&v1.Pod{}, "status.phase", helpers.FakePodStatusPhaseIndexer).
		WithObjects(objs...).
		WithStatusSubresource(&v1.Pod{}).
		Build()
}

func noop(_ context.Context, _ *v1.PodList) error { return nil }

func sidecarPodsRunning(n int) []client.Object {
	objs := make([]client.Object, n)
	for i := range objs {
		objs[i] = helpers.NewSidecarPodBuilder().
			SetName(fmt.Sprintf("pod-%d", i)).
			SetSidecarImageTag("1.11.0"). // differs from expected → will match ImageResourcesPredicate
			Build()
	}
	return objs
}

func mixedPods(n int) []client.Object {
	objs := make([]client.Object, n)
	for i := range objs {
		b := helpers.NewSidecarPodBuilder().SetName(fmt.Sprintf("pod-%d", i))
		if i%2 == 0 {
			b = b.SetSidecarImageTag("1.11.0") // needs restart
		}
		objs[i] = b.Build()
	}
	return objs
}

// BenchmarkGetPodsToRestart_NoPods measures the overhead of a single list call
// that returns no running pods at all.
func BenchmarkGetPodsToRestart_NoPods(b *testing.B) {
	ctx := context.Background()
	logger := logr.Discard()

	c := benchClient(b)
	p := pods.NewPods(c, &logger)
	preds := []predicates.SidecarProxyPredicate{
		predicates.NewImageResourcesPredicate(benchExpectedImage, helpers.DifferentSidecarResources),
	}
	limits := pods.NewPodsRestartLimits(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = p.GetPodsToRestart(ctx, preds, limits, noop)
	}
}

// BenchmarkGetPodsToRestart_AllMatch measures the path where every pod in the
// list needs a restart (all match the optional ImageResources predicate).
func BenchmarkGetPodsToRestart_AllMatch(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			c := benchClient(b, sidecarPodsRunning(n)...)
			p := pods.NewPods(c, &logger)
			preds := []predicates.SidecarProxyPredicate{
				predicates.NewImageResourcesPredicate(benchExpectedImage, helpers.DifferentSidecarResources),
			}
			limits := pods.NewPodsRestartLimits(n + 1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.GetPodsToRestart(ctx, preds, limits, noop)
			}
		})
	}
}

// BenchmarkGetPodsToRestart_NoneMatch measures the filtering path where every
// pod passes the running+sidecar filter but none match the predicate (no restart needed).
func BenchmarkGetPodsToRestart_NoneMatch(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			// Pods have the correct image tag — ImageResourcesPredicate won't match.
			objs := make([]client.Object, n)
			for i := range objs {
				objs[i] = helpers.NewSidecarPodBuilder().
					SetName(fmt.Sprintf("pod-%d", i)).
					Build()
			}
			c := benchClient(b, objs...)
			p := pods.NewPods(c, &logger)
			preds := []predicates.SidecarProxyPredicate{
				predicates.NewImageResourcesPredicate(benchExpectedImage, helpers.DefaultSidecarResources),
			}
			limits := pods.NewPodsRestartLimits(n + 1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.GetPodsToRestart(ctx, preds, limits, noop)
			}
		})
	}
}

// BenchmarkGetPodsToRestart_HalfMatch measures a realistic mixed workload where
// half the pods need a restart and half do not.
func BenchmarkGetPodsToRestart_HalfMatch(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			c := benchClient(b, mixedPods(n)...)
			p := pods.NewPods(c, &logger)
			preds := []predicates.SidecarProxyPredicate{
				predicates.NewImageResourcesPredicate(benchExpectedImage, helpers.DifferentSidecarResources),
			}
			limits := pods.NewPodsRestartLimits(n + 1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.GetPodsToRestart(ctx, preds, limits, noop)
			}
		})
	}
}

// BenchmarkGetPodsToRestart_Pagination measures the overhead of paginating
// through pods when the page size is smaller than the total pod count.
func BenchmarkGetPodsToRestart_Pagination(b *testing.B) {
	for _, n := range []int{100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d/pageSize=10", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			c := benchClient(b, sidecarPodsRunning(n)...)
			p := pods.NewPods(c, &logger)
			preds := []predicates.SidecarProxyPredicate{
				predicates.NewImageResourcesPredicate(benchExpectedImage, helpers.DifferentSidecarResources),
			}
			limits := pods.NewPodsRestartLimits(10)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.GetPodsToRestart(ctx, preds, limits, noop)
			}
		})
	}
}

// BenchmarkGetPodsToRestart_MultiplePreds measures the predicate evaluation loop
// with both a MustMatch and an optional predicate active simultaneously.
func BenchmarkGetPodsToRestart_MultiplePreds(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			c := benchClient(b, sidecarPodsRunning(n)...)
			p := pods.NewPods(c, &logger)
			preds := []predicates.SidecarProxyPredicate{
				predicates.NewImageResourcesPredicate(benchExpectedImage, helpers.DifferentSidecarResources), // optional
				predicates.CustomerWorkloadRestartPredicate{},                                                // MustMatch
			}
			limits := pods.NewPodsRestartLimits(n + 1)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = p.GetPodsToRestart(ctx, preds, limits, noop)
			}
		})
	}
}

// BenchmarkGetAllInjectedPods measures listing and filtering all pods that
// contain the istio-proxy sidecar container.
func BenchmarkGetAllInjectedPods(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			c := benchClient(b, sidecarPodsRunning(n)...)
			p := pods.NewPods(c, &logger)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = p.GetAllInjectedPods(ctx)
			}
		})
	}
}

// BenchmarkGetAllInjectedPods_Mixed measures GetAllInjectedPods when only half
// the pods have the sidecar, exercising the containsSidecar filter.
func BenchmarkGetAllInjectedPods_Mixed(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		n := n
		b.Run(fmt.Sprintf("pods=%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := logr.Discard()

			objs := make([]client.Object, n)
			for i := range objs {
				if i%2 == 0 {
					objs[i] = helpers.NewSidecarPodBuilder().SetName(fmt.Sprintf("pod-%d", i)).Build()
				} else {
					objs[i] = helpers.FixPodWithoutSidecar(fmt.Sprintf("pod-%d", i), "custom")
				}
			}
			c := benchClient(b, objs...)
			p := pods.NewPods(c, &logger)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, _ = p.GetAllInjectedPods(ctx)
			}
		})
	}
}
