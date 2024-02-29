package steps

import (
	"context"
	"fmt"

	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/pkg/manifestprocessor"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	v1 "k8s.io/api/apps/v1"
	v1c "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeployIstioOperator(ctx context.Context, manifestType, should string) error {
	var manifestFileName string
	switch manifestType {
	case "generated":
		manifestFileName = "operator_generated_manifest.yaml"
	case "failing":
		manifestFileName = "operator_failing_manifest.yaml"
	default:
		return fmt.Errorf("unsupported manifest type: %s", manifestType)
	}

	expectSuccess := false
	if should == "succeed" {
		expectSuccess = true
	}

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	resources, err := manifestprocessor.ParseYamlFromFile(manifestFileName)
	if err != nil {
		return err
	}

	return retry.Do(func() error {
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
				if expectSuccess {
					return err
				} else {
					return nil
				}
			}
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
	}, testcontext.GetRetryOpts()...)
}
