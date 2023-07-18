package steps

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/pkg/manifestprocessor"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	v1 "k8s.io/api/apps/v1"
	v1c "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeployIstioOperatorFromLocalManifest(ctx context.Context) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	resources, err := manifestprocessor.ParseFromFileWithTemplate("local-manifest.yaml", "manifests", manifestprocessor.ResourceSeparator)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
		for _, resource := range resources {
			gvk := resource.GroupVersionKind()
			var existingResource unstructured.Unstructured
			existingResource.SetGroupVersionKind(gvk)

			err := k8sClient.Get(context.TODO(), client.ObjectKey{
				Namespace: resource.GetNamespace(),
				Name:      resource.GetName(),
			}, &existingResource)

			if err != nil {
				err := k8sClient.Create(context.TODO(), &resource)
				if err != nil {
					return err
				}
			}

			mergedResource := mergeResources(&existingResource, &resource)

			err = k8sClient.Update(context.TODO(), mergedResource)
			if err != nil {
				return err
			}
		}

		var controller v1.Deployment
		err = k8sClient.Get(context.TODO(), client.ObjectKey{
			Namespace: "kyma-system",
			Name:      "istio-controller-manager",
		}, &controller)
		if err != nil {
			return err
		}
		newImage := controller.Spec.Template.Spec.Containers[0].Image

		var pods v1c.PodList
		err = k8sClient.List(context.TODO(), &pods, client.MatchingLabels{
			"app.kubernetes.io/component": "istio-operator.kyma-project.io",
		})
		if err != nil {
			return err
		}

		for _, pod := range pods.Items {
			for _, c := range pod.Spec.Containers {
				if c.Image != newImage {
					println(c.Image, ":", newImage)
					return fmt.Errorf("controller is not updated")
				}
			}
		}
		return nil
	}, testcontext.GetRetryOpts()...)
}

func mergeResources(old, new *unstructured.Unstructured) *unstructured.Unstructured {
	oldMap := old.Object
	newMap := new.Object
	mergeMaps(oldMap, newMap)
	return &unstructured.Unstructured{Object: oldMap}
}

func mergeMaps(o, n map[string]any) {
	for k, nv := range n {
		if ov, ok := o[k]; ok {
			ovm, oldIsMap := ov.(map[string]any)
			nvm, newIsMap := nv.(map[string]any)
			if oldIsMap && newIsMap {
				mergeMaps(ovm, nvm)
			} else {
				o[k] = nv
			}
		} else {
			o[k] = nv
		}
	}
}
