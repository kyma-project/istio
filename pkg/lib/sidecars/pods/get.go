package pods

import (
	"context"
	"fmt"

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
	PodsPerPage int
}

func NewPodsRestartLimits(podsPerPage int) *RestartLimits {
	return &RestartLimits{
		PodsPerPage: podsPerPage,
	}
}

type Getter interface {
	GetPodsToRestart(ctx context.Context, preds []predicates.SidecarProxyPredicate, limits *RestartLimits, restartFn func(context.Context, *v1.PodList) error) error
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
func (p *Pods) GetPodsToRestart(ctx context.Context, preds []predicates.SidecarProxyPredicate, limits *RestartLimits, restartFn func(context.Context, *v1.PodList) error) error {
	continueToken := ""

	for {
		podsWithSidecar, err := getSidecarPods(ctx, p.k8sClient, p.logger, limits.PodsPerPage, continueToken)
		if err != nil {
			return err
		}

		page := &v1.PodList{}
		for _, pod := range podsWithSidecar.Items {
			optionalMatched := false
			requiredMatched := true
			for _, predicate := range preds {
				matched := predicate.Matches(pod)
				if predicate.MustMatch() { // all of MustMatch predicates must evaluate to true
					if matched {
						p.logger.V(1).Info(fmt.Sprintf("Pod %s matches MustMatch predicate %s", pod.Name, predicate.Name()))
					} else {
						requiredMatched = false
						break
					}

				} else if !optionalMatched && matched { // at least one optional predicate must evaluate to true
					p.logger.V(1).Info(fmt.Sprintf("Pod %s matches not MustMatch predicate %s", pod.Name, predicate.Name()))
					optionalMatched = true
				}
			}
			if requiredMatched && optionalMatched {
				page.Items = append(page.Items, pod)
			}
		}

		p.logger.Info("Pods to restart on this page", "number of pods", len(page.Items))
		if len(page.Items) > 0 {
			if err := restartFn(ctx, page); err != nil {
				return err
			}
		}

		continueToken = podsWithSidecar.Continue
		if continueToken == "" {
			break
		}
	}

	return nil
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
	// Exclude pods with label istio=ingressgateway or istio=egressgateway
	// These pods are not meant to be restarted by this part of the code
	// This function is used only for restart of the the user workloads during uninstalling Istio, so the sidecars are removed
	if val, ok := pod.Labels["istio"]; ok && (val == "ingressgateway" || val == "egressgateway") {
		return false
	}
	for _, container := range pod.Spec.Containers {
		if container.Name == istioSidecarContainerName {
			return true
		}
	}
	for _, initContainer := range pod.Spec.InitContainers {
		if initContainer.Name == istioSidecarContainerName {
			return true
		}
	}
	return false
}
