package modules

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	yaml3 "gopkg.in/yaml.v3"
	v1 "k8s.io/api/apps/v1"
	v1c "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func UpgradeIstioModule(ctx context.Context, k8sClient client.Client) error {
	resources, err := parseYamlFromFile("operator_generated_manifest.yaml")
	if err != nil {
		return err
	}

	for _, resource := range resources {
		gvk := resource.GroupVersionKind()
		var existingResource unstructured.Unstructured
		existingResource.SetGroupVersionKind(gvk)

		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: resource.GetNamespace(),
			Name:      resource.GetName(),
		}, &existingResource)

		if err != nil {
			if errors.IsNotFound(err) {
				err := k8sClient.Create(ctx, &resource)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
		resource.SetResourceVersion(existingResource.GetResourceVersion())
		err = k8sClient.Update(ctx, &resource)
		if err != nil {
			return err
		}
	}

	// Wait for the deployment to be ready
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		var controller v1.Deployment
		err := k8sClient.Get(ctx, client.ObjectKey{
			Namespace: "kyma-system",
			Name:      "istio-controller-manager",
		}, &controller)
		if err != nil {
			return false, err
		}

		// Check if deployment is ready
		if controller.Status.UpdatedReplicas == *controller.Spec.Replicas &&
			controller.Status.Replicas == *controller.Spec.Replicas &&
			controller.Status.AvailableReplicas == *controller.Spec.Replicas &&
			controller.Status.ReadyReplicas == *controller.Spec.Replicas &&
			controller.Status.ObservedGeneration >= controller.Generation {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("deployment did not become ready: %w", err)
	}

	var controller v1.Deployment
	err = k8sClient.Get(ctx, client.ObjectKey{
		Namespace: "kyma-system",
		Name:      "istio-controller-manager",
	}, &controller)
	if err != nil {
		return err
	}
	newImage := controller.Spec.Template.Spec.Containers[0].Image

	var pods v1c.PodList
	err = k8sClient.List(ctx, &pods, client.MatchingLabels{
		"app.kubernetes.io/component": "istio-operator.kyma-project.io",
	})
	if err != nil {
		return err
	}

	for _, pod := range pods.Items {
		for _, c := range pod.Spec.Containers {
			if c.Image != newImage {
				return fmt.Errorf("controller is not updated")
			}
		}
	}
	return nil
}

func parseYamlFromFile(fileName string) ([]unstructured.Unstructured, error) {
	rawData, err := os.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	var manifests []unstructured.Unstructured
	decoder := yaml3.NewDecoder(bytes.NewBufferString(string(rawData)))
	for {
		var d map[string]interface{}
		if err := decoder.Decode(&d); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("document decode failed: %w", err)
		}
		manifests = append(manifests, unstructured.Unstructured{Object: d})
	}
	return manifests, nil
}
