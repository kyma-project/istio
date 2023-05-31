package pods

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/kyma-project/istio/operator/pkg/lib/sidecars/retry"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	istioValidationContainerName string = "istio-validation"
	istioInitContainerName       string = "istio-init"
	istioSidecarContainerName    string = "istio-proxy"
)

type SidecarImage struct {
	Repository string
	Tag        string
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

func getNamespacesWithIstioLabelsAndInjectionDisabled(ctx context.Context, c client.Client) (*v1.NamespaceList, *v1.NamespaceList, error) {
	unfilteredLabeledList := &v1.NamespaceList{}
	labeledList := &v1.NamespaceList{}
	disabledList := &v1.NamespaceList{}

	err := retry.RetryOnError(retry.DefaultRetry, func() error {
		return c.List(ctx, unfilteredLabeledList, client.HasLabels{"istio-injection"})
	})
	if err != nil {
		return labeledList, disabledList, err
	}

	unfilteredLabeledList.DeepCopyInto(labeledList)
	labeledList.Items = []v1.Namespace{}
	unfilteredLabeledList.DeepCopyInto(disabledList)
	disabledList.Items = []v1.Namespace{}

	for _, namespace := range unfilteredLabeledList.Items {
		if isSystemNamespace(namespace.ObjectMeta.Name) {
			continue
		}
		if namespace.Labels["istio-injection"] == "disabled" {
			disabledList.Items = append(disabledList.Items, namespace)
		}
		labeledList.Items = append(labeledList.Items, namespace)
	}

	return labeledList, disabledList, err
}

func GetPodsToRestart(ctx context.Context, c client.Client, expectedImage SidecarImage, expectedResources v1.ResourceRequirements, logger *logr.Logger) (outputPodsList v1.PodList, err error) {
	podList, err := getAllRunningPods(ctx, c)
	if err != nil {
		return outputPodsList, err
	}

	podList.DeepCopyInto(&outputPodsList)
	outputPodsList.Items = []v1.Pod{}

	for _, pod := range podList.Items {
		if needsRestart(pod, expectedImage, expectedResources) {
			outputPodsList.Items = append(outputPodsList.Items, *pod.DeepCopy())
		}
	}

	logger.Info("Pods to restart", "number of pods", len(outputPodsList.Items))
	return outputPodsList, nil
}

func GetPodsForCNIChange(ctx context.Context, c client.Client, isCNIEnabled bool, logger *logr.Logger) (outputPodsList v1.PodList, err error) {
	podList, err := getAllRunningPods(ctx, c)
	if err != nil {
		return outputPodsList, err
	}

	var containerName string
	if isCNIEnabled {
		containerName = istioInitContainerName
	} else {
		containerName = istioValidationContainerName
	}

	_, injectionDisabledNamespaceList, err := getNamespacesWithIstioLabelsAndInjectionDisabled(ctx, c)
	if err != nil {
		return outputPodsList, err
	}

	podList.DeepCopyInto(&outputPodsList)
	outputPodsList.Items = []v1.Pod{}

	for _, pod := range podList.Items {
		if isPodReady(pod) && hasInitContainer(pod.Spec.InitContainers, containerName) &&
			!isPodInNamespaceList(pod, injectionDisabledNamespaceList.Items) &&
			!isSystemNamespace(pod.Namespace) {
			outputPodsList.Items = append(outputPodsList.Items, *pod.DeepCopy())
		}
	}

	logger.Info("Pods that need to adapt to CNI change", "number of pods", len(outputPodsList.Items))

	return outputPodsList, nil
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
