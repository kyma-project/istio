package setup

import (
	"context"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/config"
)

func ShouldSkipCleanup(t *testing.T) bool {
	return t.Failed() && config.Get().SkipCleanup
}

func DeclareCleanup(t *testing.T, f func()) {
	t.Helper()
	t.Cleanup(func() {
		t.Helper()
		DumpClusterResources(t)
		if ShouldSkipCleanup(t) {
			t.Logf("Tests failed, skipping cleanup")
			return
		}
		t.Logf("Cleaning up")
		f()
	})
}

func GetCleanupContext() context.Context {
	return context.Background()
}
