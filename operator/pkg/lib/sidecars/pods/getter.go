package pods

import (
	"context"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type SidecarImage struct {
	Repository string
	Tag        string
}

const (
	istioValidationContainerName = "istio-validation"
	istioInitContainerName       = "istio-init"
	istioSidecarName             = "istio-proxy"
)

func (r SidecarImage) matchesImageIn(container v1.Container) bool {
	// TODO Understand why we can do a full string match
	containsRepository := strings.Contains(container.Image, r.Repository)
	containsTag := strings.HasSuffix(container.Image, r.Tag)
	return containsRepository && containsTag
}

func getAllRunningPods(ctx context.Context, c client.Client) (*v1.PodList, error) {
	podList := &v1.PodList{}

	isRunning := fields.OneTermEqualSelector("status.phase", string(v1.PodRunning))

	err := c.List(ctx, podList, client.MatchingFieldsSelector{Selector: isRunning})
	if err != nil {
		return podList, err
	}

	return podList, nil
}

func getNamespacesWithIstioInjection(ctx context.Context, c client.Client) (*v1.NamespaceList, error) {
	allNamespaceList := &v1.NamespaceList{}
	istioInjectionNamespaceList := &v1.NamespaceList{}

	err := c.List(ctx, allNamespaceList, client.MatchingLabels{"istio-injection": "enabled"})
	if err != nil {
		return istioInjectionNamespaceList, err
	}

	allNamespaceList.DeepCopyInto(istioInjectionNamespaceList)
	istioInjectionNamespaceList.Items = []v1.Namespace{}

	for _, namespace := range allNamespaceList.Items {
		switch namespace.ObjectMeta.Name {
		case "kube-system":
			continue
		case "kube-public":
			continue
		case "istio-system":
			continue
		default:
			istioInjectionNamespaceList.Items = append(istioInjectionNamespaceList.Items, namespace)
		}
	}

	return istioInjectionNamespaceList, err
}

func GetPodsWithDifferentSidecarImage(ctx context.Context, c client.Client, expectedImage SidecarImage) (outputPodsList v1.PodList, err error) {
	podList, err := getAllRunningPods(ctx, c)
	if err != nil {
		return outputPodsList, err
	}

	podList.DeepCopyInto(&outputPodsList)
	outputPodsList.Items = []v1.Pod{}

	for _, pod := range podList.Items {
		if hasIstioSidecarStatusAnnotation(pod) &&
			isPodReady(pod) &&
			hasSidecarContainerWithWithDifferentImage(pod, expectedImage) {
			outputPodsList.Items = append(outputPodsList.Items, *pod.DeepCopy())
		}
	}

	return outputPodsList, nil
}

func GetPodsForCNIChange(ctx context.Context, c client.Client, isCNIEnabled bool) (outputPodsList v1.PodList, err error) {
	podList, err := getAllRunningPods(ctx, c)
	// TODO add logs
	if err != nil {
		return outputPodsList, err
	}

	var containerName string
	if isCNIEnabled {
		containerName = istioInitContainerName
	} else {
		containerName = istioValidationContainerName
	}

	injectionEnabledNamespaceList, err := getNamespacesWithIstioInjection(ctx, c)
	if err != nil {
		return outputPodsList, err
	}

	podList.DeepCopyInto(&outputPodsList)
	outputPodsList.Items = []v1.Pod{}

	for _, pod := range podList.Items {
		if isPodReady(pod) && hasInitContainer(pod.Spec.InitContainers, containerName) &&
			// TODO: Fix checking only pods in istio namespaces and filter out kube-system ns pods
			isPodInNamespaceList(pod, injectionEnabledNamespaceList.Items) {
			outputPodsList.Items = append(outputPodsList.Items, *pod.DeepCopy())
		}
	}

	return outputPodsList, nil
}

func GetPodsWithoutSidecar(ctx context.Context, c client.Client, isSidecarInjectionEnabledByDefault bool) (outputPodsList v1.PodList, err error) {
	podList, err := getAllRunningPods(ctx, c)
	// TODO add logs
	if err != nil {
		return outputPodsList, err
	}

	injectionEnabledNamespaceList, err := getNamespacesWithIstioInjection(ctx, c)
	if err != nil {
		return outputPodsList, err
	}

	podList.DeepCopyInto(&outputPodsList)
	outputPodsList.Items = []v1.Pod{}

	for _, pod := range podList.Items {
		if isPodReady(pod) &&
			!hasIstioSidecarContainer(pod.Spec.Containers, istioSidecarName) &&
			// TODO: Fix checking only pods in istio namespaces and filter out kube-system ns pods
			isPodInNamespaceList(pod, injectionEnabledNamespaceList.Items) {
			outputPodsList.Items = append(outputPodsList.Items, *pod.DeepCopy())
		}
	}

	return outputPodsList, nil
}
