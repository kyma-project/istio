package steps

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const applicationImage = "europe-docker.pkg.dev/kyma-project/prod/external/kennethreitz/httpbin"

func CreateApplicationDeployment(ctx context.Context, appName, namespace string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	dep := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
		},
		Spec: v1.DeploymentSpec{
			Replicas: pointer.Int32(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": appName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": appName},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  appName,
							Image: applicationImage,
						},
					},
				},
			},
		},
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &dep)
		if err != nil {
			return err
		}
		ctx = testcontext.SetTestAppInContext(ctx, &dep)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func ApplicationHasProxyResourcesSetToCpuAndMemory(ctx context.Context, appName, appNamespace, resourceType, cpu, memory string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	var podList corev1.PodList
	return retry.Do(func() error {
		err := k8sClient.List(context.TODO(), &podList, &client.ListOptions{
			Namespace: appNamespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": appName,
			}),
		})
		if err != nil {
			return err
		}

		if len(podList.Items) == 0 {
			return fmt.Errorf("no pods found for app %s in namespace %s", appName, appNamespace)
		}

		hasResources := false
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				if container.Name != "istio-proxy" {
					continue
				}

				switch resourceType {
				case "limits":
					hasResources = container.Resources.Limits.Cpu().String() == cpu &&
						container.Resources.Limits.Memory().String() == memory
				case "requests":
					hasResources = container.Resources.Requests.Cpu().String() == cpu &&
						container.Resources.Requests.Memory().String() == memory
				default:
					return fmt.Errorf("resource type %s is not supported", resourceType)

				}
			}
		}

		if !hasResources {
			return fmt.Errorf("the resources of proxy of app %s in namespace %s does not match %s cpu %s memory %s",
				appName, appNamespace, resourceType, cpu, memory)
		}

		return nil
	}, testcontext.GetRetryOpts()...)
}
