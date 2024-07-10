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

func listRunningPods(ctx context.Context, c client.Client, limit int, continueToken string) (*v1.PodList, error) {
	podList := &v1.PodList{}

	err := retry.RetryOnError(retry.DefaultRetry, func() error {
		listOps := []client.ListOption{
			client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector("status.phase", string(v1.PodRunning))},
			client.Limit(limit),
		}
		if continueToken != "" {
			listOps = append(listOps, client.Continue(continueToken))
		}
		return c.List(ctx, podList, listOps...)
	})

	return podList, err
}

func getSidecarPods(ctx context.Context, c client.Client, logger *logr.Logger, limit int, continueToken string) (*v1.PodList, error) {
	podList, err := listRunningPods(ctx, c, limit, continueToken)
	if err != nil {
		return nil, err
	}

	logger.Info("Retrieved running pods for proxy restart", "number of pods", len(podList.Items), "has more pods", podList.Continue != "")

	podsWithSidecar := &v1.PodList{}
	podsWithSidecar.Continue = podList.Continue

	for _, pod := range podList.Items {
		if isReadyWithIstioAnnotation(pod) {
			podsWithSidecar.Items = append(podsWithSidecar.Items, pod)
		}
	}

	logger.Info("Filtered pods with Istio sidecar", "number of pods", len(podsWithSidecar.Items))

	return podsWithSidecar, nil
}

func GetPodsToRestart(ctx context.Context, c client.Client, expectedImage SidecarImage, expectedResources v1.ResourceRequirements, predicates []filter.SidecarProxyPredicate, limit int, logger *logr.Logger) (*v1.PodList, error) {
	//Add predicate for image version and resources configuration
	predicates = append(predicates, NewRestartProxyPredicate(expectedImage, expectedResources))

	podsToRestart := &v1.PodList{}
	for while := true; while; {
		fmt.Print("getting pods\n")
		fmt.Printf("continue tocken beginning of loop: %v\n", podsToRestart.Continue)
		podsWithSidecar, err := getSidecarPods(ctx, c, logger, limit, podsToRestart.Continue)
		if err != nil {
			return nil, err
		}
		fmt.Print("got:\n")
		for _, pod := range podsWithSidecar.Items {
			fmt.Printf("pod: %s\n", pod.Name)
		}
		for _, pod := range podsWithSidecar.Items {
			for _, predicate := range predicates {
				if predicate.RequiresProxyRestart(pod) {
					podsToRestart.Items = append(podsToRestart.Items, pod)
					break
				}
			}
		}
		fmt.Print("needs to restart\n")
		for _, pod := range podsToRestart.Items {
			fmt.Printf("pod: %s\n", pod.Name)
		}
		podsToRestart.Continue = podsWithSidecar.Continue
		fmt.Printf("continue tocken end of loop: %v\n", podsToRestart.Continue)
		while = len(podsToRestart.Items) < limit && podsToRestart.Continue != ""
	}

	if len(podsToRestart.Items) > 0 {
		logger.Info("Pods to restart", "number of pods", len(podsToRestart.Items), "has more pods", podsToRestart.Continue != "")
	}

	return podsToRestart, nil
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
