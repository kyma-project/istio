package sidecars

import (
	"context"
	"github.com/kyma-project/istio/operator/api/v1alpha1"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/pods"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/restart"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ProxyReset(ctx context.Context, c client.Client, expectedImage pods.SidecarImage, expectedResources v1alpha1.Resources, cniEnabled bool, logger *logr.Logger) ([]restart.RestartWarning, error) {
	differentImagePodList, err := pods.GetPodsToRestart(ctx, c, expectedImage, expectedResources, logger)
	if err != nil {
		return nil, err
	}

	cniPodList, err := pods.GetPodsForCNIChange(ctx, c, cniEnabled, logger)
	if err != nil {
		return nil, err
	}

	var podListToRestart v1.PodList
	podListToRestart.Items = []v1.Pod{}
	differentImagePodList.DeepCopyInto(&podListToRestart)
	podListToRestart.Items = append(podListToRestart.Items, cniPodList.DeepCopy().Items...)

	warnings, err := restart.Restart(ctx, c, podListToRestart, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("Proxy reset done")

	return warnings, nil

}
