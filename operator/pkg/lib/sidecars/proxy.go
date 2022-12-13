package sidecars

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//
//func Reset(c client.Client, logger logr.Logger) error {
//	ctx := context.Background()
//	// TODO: Check for CNI and sidecar injection enabled by default?
//
//	istioPods, err := pods.GetAllIstioPods(ctx, c, logger)
//	if err != nil {
//		return fmt.Errorf("failed to get all pods: %v", err)
//	}
//
//	logger.Info(fmt.Sprintln(istioPods))
//
//	//cfg := IstioProxyConfig{
//	//	Context:                          ctx,
//	//	ImagePrefix:                      proxyImagePrefix,
//	//	ImageVersion:                     proxyImageVersion,
//	//	Kubeclient:                       kubeClient,
//	//	Debug:                            false,
//	//	Log:                              logger,
//	//	SidecarInjectionByDefaultEnabled: sidecarInjectionEnabledByDefault,
//	//	CNIEnabled:                       cniEnabled,
//	//}
//
//	return nil
//}

func ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, namespaceEnabledByDefault, cniEnabled bool, logger *logr.Logger) ([]restart.RestartWarning, error) {
	differentImagePodList, err := pods.GetPodsWithDifferentSidecarImage(ctx, c, expectedImage)
	if err != nil {
		return nil, err
	}

	noSidecarPodList, err := pods.GetPodsWithoutSidecar(ctx, c, namespaceEnabledByDefault)
	if err != nil {
		return nil, err
	}

	cniPodList, err := pods.GetPodsForCNIChange(ctx, c, cniEnabled)
	if err != nil {
		return nil, err
	}

	var podListToRestart v1.PodList
	podListToRestart.Items = []v1.Pod{}
	differentImagePodList.DeepCopyInto(&podListToRestart)
	podListToRestart.Items = append(podListToRestart.Items, noSidecarPodList.DeepCopy().Items...)
	podListToRestart.Items = append(podListToRestart.Items, cniPodList.DeepCopy().Items...)

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

	warnings, err := restart.Restart(ctx, c, podListToRestart)
	if err != nil {
		return nil, err
	}

	logger.V(2).Info("proxy reset for successfully done")

	return warnings, nil

}
