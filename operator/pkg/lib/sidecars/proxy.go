package sidecars

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, namespaceEnabledByDefault, cniEnabled bool, logger logr.Logger) ([]restart.RestartWarning, error) {
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
		// TODO Decide if have to keep the reset warning filter based on annotation
		podListToRestart.Items = append(podListToRestart.Items, pod)
	}

	warnings, err := restart.Restart(ctx, c, podListToRestart)
	if err != nil {
		return nil, err
	}

	logger.Info("proxy reset for successfully done")

	return warnings, nil

}
