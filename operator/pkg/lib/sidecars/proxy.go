package sidecars

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	v1 "k8s.io/api/core/v1"
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

func RestartPodWithDifferentSidecarImage(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, logger *logr.Logger) error {
	differentImagePodList, err := pods.GetPodsWithDifferentSidecarImage(ctx, c, expectedImage)
	if err != nil {
		return err
	}

	var podListToRestart v1.PodList
	differentImagePodList.DeepCopyInto(&podListToRestart)
	podListToRestart.Items = []v1.Pod{}

	for _, pod := range differentImagePodList.Items {
		// We need to skip pods with reset warning as they can't be restarted for some reason.
		if pods.HasResetWarning(pod) {
			logger.V(1).Info("found pod with different istio proxy image that can't be updated.",
				"pod name", pod.Name, "pod namespace", pod.Namespace)
			continue
		}

		// TODO Decide if have to keep the reset warning filter based on annotation
		podListToRestart.Items = append(podListToRestart.Items, pod)
	}

	err = reset(ctx, c, podListToRestart)
	if err != nil {
		return err
	}

	logger.V(2).Info("proxy reset for successfully done")

	return nil

}

func reset(ctx context.Context, c client.Client, podList v1.PodList) error {
	return nil
}
