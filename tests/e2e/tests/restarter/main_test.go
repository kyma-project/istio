package restarter

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/kyma-project/istio/operator/tests/e2e/pkg/config"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/client"
	"github.com/kyma-project/istio/operator/tests/e2e/pkg/helpers/modules"
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	r, err := client.ResourcesClientFromConfig()
	if err != nil {
		log.Fatalf("restarter suite: client: %v", err)
	}
	icr, err := modules.InstallDefaultIstio(ctx, r)
	if err != nil {
		log.Fatalf("restarter suite: install istio: %v", err)
	}
	code := m.Run()
	if !config.Get().SkipCleanup {
		if err := modules.UninstallIstio(ctx, r, icr); err != nil {
			log.Printf("restarter suite: uninstall istio: %v", err)
		}
	}
	os.Exit(code)
}
