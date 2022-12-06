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
	requiredNamespaceList := &v1.NamespaceList{}

	err := c.List(ctx, allNamespaceList, client.MatchingLabels{"istio-injection": "enabled"})
	if err != nil {
		return requiredNamespaceList, err
	}

	allNamespaceList.DeepCopyInto(requiredNamespaceList)
	requiredNamespaceList.Items = []v1.Namespace{}

	for _, namespace := range allNamespaceList.Items {
		switch namespace.ObjectMeta.Name {
		case "kube-system":
			continue
		case "kube-public":
			continue
		case "istio-system":
			continue
		default:
			requiredNamespaceList.Items = append(requiredNamespaceList.Items, namespace)
		}
	}

	return requiredNamespaceList, err
}

func GetPodsWithDifferentSidecarImage(ctx context.Context, c client.Client, expectedImage SidecarImage) (outputPodsList v1.PodList, err error) {
	podList, err := getAllRunningPods(ctx, c)
	// TODO add logs
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

func GetPodsForCNIChange(ctx context.Context, c client.Client, expectedImage SidecarImage) (outputPodsList v1.PodList, err error) {
	podList, err := getAllRunningPods(ctx, c)
	// TODO add logs
	if err != nil {
		return outputPodsList, err
	}

	//istioNamespaceList, err := getNamespacesWithIstioInjection(ctx, c)
	//if err != nil {
	//	return outputPodsList, err
	//}

	podList.DeepCopyInto(&outputPodsList)
	outputPodsList.Items = []v1.Pod{}

	for _, pod := range podList.Items {
		// TODO: init container name logic
		if isPodReady(pod) && hasInitContainer(pod.Spec.Containers, "istio-init") {
			outputPodsList.Items = append(outputPodsList.Items, *pod.DeepCopy())
		}
	}

	return
}

func isPodInNamespaceList(pod v1.Pod, namespaceList v1.NamespaceList) bool {
	for _, namespace := range namespaceList.Items {
		if pod.ObjectMeta.Namespace == namespace.Name {
			return true
		}
	}

	return false
}
