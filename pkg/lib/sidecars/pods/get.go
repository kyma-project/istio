package pods

import (
	"context"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kyma-project/istio/operator/internal/restarter/predicates"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
)

const (
	istioSidecarContainerName string = "istio-proxy"
)

type RestartLimits struct {
	PodsToRestartLimit int
	PodsToListLimit    int
}

func NewPodsRestartLimits(restartLimit, listLimit int) *RestartLimits {
	return &RestartLimits{
		PodsToRestartLimit: restartLimit,
		PodsToListLimit:    listLimit,
	}
}

type Getter interface {
	GetPodsToRestart(ctx context.Context, preds []predicates.SidecarProxyPredicate, limits *RestartLimits) (*v1.PodList, error)
	GetAllInjectedPods(context context.Context) (*v1.PodList, error)
}

type Pods struct {
	k8sClient client.Client
	logger    *logr.Logger
}

func NewPods(k8sClient client.Client, logger *logr.Logger) *Pods {
	return &Pods{
		k8sClient: k8sClient,
		logger:    logger,
	}
}

//nolint:gocognit // cognitive complexity 29 of func `(*Pods).GetPodsToRestart` is high (> 20) TODO refactor
func (p *Pods) GetPodsToRestart(ctx context.Context, preds []predicates.SidecarProxyPredicate, limits *RestartLimits) (*v1.PodList, error) {
	podsToRestart := &v1.PodList{}
	for while := true; while; {
		podsWithSidecar, err := getSidecarPods(ctx, p.k8sClient, p.logger, limits.PodsToListLimit, podsToRestart.Continue)
		if err != nil {
			return nil, err
		}
		for _, pod := range podsWithSidecar.Items {
			optionalMatched := false
			requiredMatched := true
			for _, predicate := range preds {
				matched := predicate.Matches(pod)
				if predicate.MustMatch() { // if predicate must match, all must match
					if !matched {
						requiredMatched = false
						break
					}
				} else if !optionalMatched && matched { // if predicate is optional, at least one must match
					optionalMatched = true
				}
			}
			if requiredMatched && optionalMatched {
				podsToRestart.Items = append(podsToRestart.Items, pod)
			}
			if len(podsToRestart.Items) >= limits.PodsToRestartLimit {
				break
			}
		}
		podsToRestart.Continue = podsWithSidecar.Continue
		while = len(podsToRestart.Items) < limits.PodsToRestartLimit && podsToRestart.Continue != ""
	}

	if len(podsToRestart.Items) > 0 {
		p.logger.Info("Pods to restart", "number of pods", len(podsToRestart.Items), "has more pods", podsToRestart.Continue != "")
	} else {
		p.logger.Info("No pods to restart with matching predicates")
	}

	return podsToRestart, nil
}

func (p *Pods) GetAllInjectedPods(ctx context.Context) (*v1.PodList, error) {
	podList := &v1.PodList{}
	outputPodList := &v1.PodList{}
	outputPodList.Items = make([]v1.Pod, len(podList.Items))

	err := retry.OnError(retry.DefaultRetry, func() error {
		return p.k8sClient.List(ctx, podList, &client.ListOptions{})
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

func listRunningPods(ctx context.Context, c client.Client, listLimit int, continueToken string) (*v1.PodList, error) {
	podList := &v1.PodList{}

	err := retry.OnError(retry.DefaultRetry, func() error {
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
