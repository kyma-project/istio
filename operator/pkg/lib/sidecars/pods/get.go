package pods

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

const AnnotationResetWarningKey = "istio.reconciler.kyma-project.io/proxy-reset-warning"

func GetAllIstioPods(ctx context.Context, c client.Client, logger logr.Logger) (*v1.PodList, error) {
	podList, err := GetIstioPodsWithSelectors(ctx, c)
	if err != nil {
		return nil, err
	}

	podsWithDifferentImage := GetIstioPodsWithDifferentImage(podList)

	// TODO: Remove this log
	logger.Info(fmt.Sprintln(podsWithDifferentImage))

	return podList, err
}

func GetIstioPodsWithSelectors(ctx context.Context, c client.Client) (*v1.PodList, error) {
	podList := &v1.PodList{}

	sidecarAnnotation := fields.OneTermEqualSelector("metadata.annotations", "sidecar.istio.io/status")
	phase := fields.OneTermEqualSelector("status.phase", string(v1.PodRunning))
	// TODO: Selector (or in naive loop) for istio sidecar names from annotations
	resetWarningAnnotation := fields.OneTermNotEqualSelector("metadata.annotations", AnnotationResetWarningKey)

	selectors := fields.AndSelectors(sidecarAnnotation, phase, resetWarningAnnotation)
	err := c.List(ctx, podList, client.MatchingFieldsSelector{Selector: selectors})
	if err != nil {
		return podList, err
	}

	return podList, nil
}

func GetIstioPodsWithDifferentImage(inputPodList *v1.PodList) (outputPodList v1.PodList) {
	inputPodList.DeepCopyInto(&outputPodList)
	outputPodList.Items = []v1.Pod{}

	tmpPrefix := ""
	tmpSuffix := ""

	// TODO: Check if it's possible to create Selector for prefix and suffix
	for _, pod := range inputPodList.Items {
		for _, container := range pod.Spec.Containers {
			containsPrefix := strings.Contains(container.Image, tmpPrefix)
			hasSuffix := strings.HasSuffix(container.Image, tmpSuffix)
			if !hasSuffix || !containsPrefix {
				outputPodList.Items = append(outputPodList.Items, *pod.DeepCopy())
			}
		}
	}

	return
}

func checkPodSidecarInjectionLogic(pod v1.Pod, sidecarInjectionEnabledByDefault bool) (requireSidecar bool) {
	namespaceLabelValue, namespaceLabeled := pod.Annotations["reconciler/namespace-istio-injection"]
	podAnnotationValue, podAnnotated := pod.Annotations["sidecar.istio.io/inject"]
	podLabelValue, podLabeled := pod.Labels["sidecar.istio.io/inject"]

	//Automatic sidecar injection is ignored for pods on the host network
	if pod.Spec.HostNetwork {
		return false
	}

	if namespaceLabeled && namespaceLabelValue == "disabled" {
		return false
	}

	if podLabeled && podLabelValue == "false" {
		return false
	}

	if !podLabeled && podAnnotated && podAnnotationValue == "false" {
		return false
	}

	if !sidecarInjectionEnabledByDefault && !namespaceLabeled && podAnnotated && podAnnotationValue == "true" {
		return false
	}

	if !sidecarInjectionEnabledByDefault && !namespaceLabeled && !podAnnotated && !podLabeled {
		return false
	}

	return true
}
