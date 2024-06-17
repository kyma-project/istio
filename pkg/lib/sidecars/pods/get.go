package pods

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/internal/filter"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	istioSidecarContainerName string = "istio-proxy"
)

type SidecarImage struct {
	Repository string
	Tag        string
}

func NewSidecarImage(hub, tag string) SidecarImage {
	return SidecarImage{
		Repository: fmt.Sprintf("%s/proxyv2", hub),
		Tag:        tag,
	}
}

func (r SidecarImage) String() string {
	return fmt.Sprintf("%s:%s", r.Repository, r.Tag)
}

func (r SidecarImage) matchesImageIn(container v1.Container) bool {
	return container.Image == r.String()
}

func getAllRunningPods(ctx context.Context, c client.Client) (*v1.PodList, error) {
	podList := &v1.PodList{}

	isRunning := fields.OneTermEqualSelector("status.phase", string(v1.PodRunning))

	err := retry.RetryOnError(retry.DefaultRetry, func() error {
		return c.List(ctx, podList, client.MatchingFieldsSelector{Selector: isRunning})
	})
	if err != nil {
		return podList, err
	}

	return podList, nil
}

func getSidecarPods(ctx context.Context, c client.Client, logger *logr.Logger) (*[]v1.Pod, error) {
	podList, err := getAllRunningPods(ctx, c)
	if err != nil {
		return nil, err
	}
	logger.Info("Read all running pods for proxy restart", "number of pods", len(podList.Items))

	podsWithSidecar := make([]v1.Pod, 0, len(podList.Items))
	for _, pod := range podList.Items {
		if isReadyWithIstioAnnotation(pod) {
			podsWithSidecar = append(podsWithSidecar, pod)
		}
	}
	logger.Info("Filtered pods with Istio sidecar", "number of pods", len(podsWithSidecar))

	return &podsWithSidecar, nil
}

func GetPodsToRestart(ctx context.Context, c client.Client, expectedImage SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, logger *logr.Logger) (outputPodsList *v1.PodList, err error) {
	//Add predicate for image version and resources configuration
	predicates = append(predicates, NewRestartProxyPredicate(expectedImage, expectedResources))
	restartEvaluator, err := initRestartEvaluator(ctx, predicates)
	if err != nil {
		return &v1.PodList{}, err
	}

	pods, err := getSidecarPods(ctx, c, logger)
	if err != nil {
		return &v1.PodList{}, err
	}

	outputPodsList = &v1.PodList{}
	outputPodsList.Items = make([]v1.Pod, 0, len(*pods))

	for _, pod := range *pods {
		for _, evaluator := range restartEvaluator {
			if evaluator.RequiresProxyRestart(pod) {
				outputPodsList.Items = append(outputPodsList.Items, pod)
				break
			}
		}
	}

	if outputPodsList != nil {
		logger.Info("Pods to restart", "number of pods", len(outputPodsList.Items))
	}
	return outputPodsList, nil
}

func initRestartEvaluator(ctx context.Context, predicates []filter.SidecarProxyPredicate) ([]filter.ProxyRestartEvaluator, error) {
	var evaluators []filter.ProxyRestartEvaluator

	for _, predicate := range predicates {
		e, err := predicate.NewProxyRestartEvaluator(ctx)
		if err != nil {
			return nil, err
		}

		evaluators = append(evaluators, e)
	}
	return evaluators, nil
}

func containsSidecar(pod v1.Pod) bool {
	// If the pod has one container it is not injected
	// This skips IngressGateway pods, as those only have istio-proxy
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
