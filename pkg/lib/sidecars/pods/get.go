package pods

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	istioSidecarContainerName string = "istio-proxy"
)

type PodsRestartLimits struct {
	podsToRestartLimit int
	podsToListLimit    int
}

func NewPodsRestartLimits(restartLimit, listLimit int) *PodsRestartLimits {
	return &PodsRestartLimits{
		podsToRestartLimit: restartLimit,
		podsToListLimit:    listLimit,
	}
}

func listRunningPods(ctx context.Context, c client.Client, listLimit int, continueToken string) (*v1.PodList, error) {
	podList := &v1.PodList{}

	err := retry.RetryOnError(retry.DefaultRetry, func() error {
		listOps := []client.ListOption{
			client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning))},
			client.Limit(listLimit),
		}
		if continueToken != "" {
			listOps = append(listOps, client.Continue(continueToken))
		}
		return c.List(ctx, podList, listOps...)
	})

	return podList, err
}

func getSidecarPods(ctx context.Context, c client.Client, logger *logr.Logger, listLimit int, continueToken string) (*v1.PodList, error) {
	podList, err := listRunningPods(ctx, c, listLimit, continueToken)
	if err != nil {
		return nil, err
	}

	logger.Info("Got running pods for proxy restart", "number of pods", len(podList.Items), "has more pods", podList.Continue != "")

	podsWithSidecar := &v1.PodList{}
	podsWithSidecar.Continue = podList.Continue

	for _, pod := range podList.Items {
		if predicates.IsReadyWithIstioAnnotation(pod) {
			podsWithSidecar.Items = append(podsWithSidecar.Items, pod)
		}
	}

	logger.Info("Filtered pods with Istio sidecar", "number of pods", len(podsWithSidecar.Items))
	return podsWithSidecar, nil
}

func GetPodsToRestart(ctx context.Context, c client.Client, expectedImage predicates.SidecarImage, expectedResources v1.ResourceRequirements, preds []predicates.SidecarProxyPredicate, limits *PodsRestartLimits, logger *logr.Logger) (*v1.PodList, error) {
	preds = append(preds, predicates.NewImageResourcesPredicate(expectedImage, expectedResources))

	podsToRestart := &v1.PodList{}
	for while := true; while; {
		podsWithSidecar, err := getSidecarPods(ctx, c, logger, limits.podsToListLimit, podsToRestart.Continue)
		if err != nil {
			return nil, err
		}
		for _, pod := range podsWithSidecar.Items {
			for _, predicate := range preds { // any predicate match will trigger a restart
				if predicate.RequiresProxyRestart(pod) {
					podsToRestart.Items = append(podsToRestart.Items, pod)
					break
				}
			}
			if len(podsToRestart.Items) >= limits.podsToRestartLimit {
				break
			}
		}
		podsToRestart.Continue = podsWithSidecar.Continue
		while = len(podsToRestart.Items) < limits.podsToRestartLimit && podsToRestart.Continue != ""
	}

	if len(podsToRestart.Items) > 0 {
		logger.Info("Pods to restart", "number of pods", len(podsToRestart.Items), "has more pods", podsToRestart.Continue != "")
	}

	return podsToRestart, nil
}

func containsSidecar(pod v1.Pod) bool {
	// If the pod has one container it is not injected
	// This skips IngressGateway and EgressGateway pods, as those only have istio-proxy
	if len(pod.Spec.Containers) == 1 {
		return false
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == istioSidecarContainerName {
			return true
		}
	}
	return false
}

func GetAllInjectedPods(ctx context.Context, k8sclient client.Client) (outputPodList *v1.PodList, err error) {
	podList := &v1.PodList{}
	outputPodList = &v1.PodList{}
	outputPodList.Items = make([]v1.Pod, len(podList.Items))

	err = retry.RetryOnError(retry.DefaultRetry, func() error {
		return k8sclient.List(ctx, podList, &client.ListOptions{})
	})
	if err != nil {
		return podList, err
	}

	for _, pod := range podList.Items {
		if containsSidecar(pod) {
			outputPodList.Items = append(outputPodList.Items, pod)
		}
	}

	return outputPodList, nil
}
