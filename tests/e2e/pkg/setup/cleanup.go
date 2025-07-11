package setup

import (
	"context"
	"os"
	"testing"
)

var shouldSkipCleanup = os.Getenv("SKIP_CLEANUP") == "true"

func ShouldSkipCleanup(t *testing.T) bool {
	return t.Failed() && shouldSkipCleanup
}

func DeclareCleanup(t *testing.T, f func()) {
	t.Helper()
	t.Cleanup(func() {
		t.Helper()
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
