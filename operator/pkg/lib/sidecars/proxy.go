package sidecars

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Reset(c client.Client, logger logr.Logger) error {
	ctx := context.Background()
	// TODO: Check for CNI and sidecar injection enabled by default?

	istioPods, err := pods.GetAllIstioPods(ctx, c, logger)
	if err != nil {
		return fmt.Errorf("failed to get all pods: %v", err)
	}

	logger.Info(fmt.Sprintln(istioPods))

	//cfg := IstioProxyConfig{
	//	Context:                          ctx,
	//	ImagePrefix:                      proxyImagePrefix,
	//	ImageVersion:                     proxyImageVersion,
	//	Kubeclient:                       kubeClient,
	//	Debug:                            false,
	//	Log:                              logger,
	//	SidecarInjectionByDefaultEnabled: sidecarInjectionEnabledByDefault,
	//	CNIEnabled:                       cniEnabled,
	//}

	return nil
}

func DeleteSidecarWithDifferentImage(ctx context.Context, c client.Client, expectedImage pods.SidecarImage) {
	pods.GetPodsWithDifferentSidecarImage(ctx, c, expectedImage)
}
