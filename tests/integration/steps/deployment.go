package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/avast/retry-go"
	"github.com/distribution/reference"
	"github.com/kyma-project/istio/operator/controllers"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/masterminds/semver"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	httpbinImage  = "europe-docker.pkg.dev/kyma-project/prod/external/kennethreitz/httpbin"
	extAuthzImage = "gcr.io/istio-testing/ext-authz:latest"
)

func CreateDeployment(ctx context.Context, appName, namespace string, container corev1.Container, volumes ...corev1.Volume) (context.Context, error) {
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
			Replicas: ptr.To(int32(1)),
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
						container,
					},
				},
			},
		},
	}

	if len(volumes) > 0 {
		dep.Spec.Template.Spec.Volumes = volumes
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &dep)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, &dep)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func CreateApplicationDeployment(ctx context.Context, appName, image, namespace string) (context.Context, error) {
	c := corev1.Container{
		Name:  appName,
		Image: image,
	}

	return CreateDeployment(ctx, appName, namespace, c)
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

// ApplicationPodShouldHaveIstioProxy checks depending on the shouldBePresent parameter if the pod has an istio-proxy container.
// If shouldBePresent is "present", it checks if the pod has istio-proxy container, any other string checks if the pod does not have istio-proxy container.
func ApplicationPodShouldHaveIstioProxy(ctx context.Context, appName, namespace, shouldBePresent string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	shouldHaveProxy := shouldBePresent == "present"

	var podList corev1.PodList
	return retry.Do(func() error {
		podListOpts := &client.ListOptions{
			Namespace: namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": appName,
			}),
		}
		err := getPodList(ctx, k8sClient, &podList, podListOpts)
		if err != nil {
			return err
		}

		if len(podList.Items) == 0 {
			return fmt.Errorf("no pods found for app %s in namespace %s", appName, namespace)
		}

		hasProxy := false
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				if container.Name == "istio-proxy" {
					hasProxy = true
				}
			}

			switch {
			case shouldHaveProxy && hasProxy:
				return nil
			case !shouldHaveProxy && !hasProxy:
				return nil
			case shouldHaveProxy && !hasProxy:
				return fmt.Errorf("the pod %s in namespace %s does not have istio-proxy", pod.Name, pod.Namespace)
			case !shouldHaveProxy && hasProxy:
				return fmt.Errorf("the pod %s in namespace %s has istio-proxy", pod.Name, pod.Namespace)
			default:
				return fmt.Errorf("the pod %s in namespace %s has unexpected istio-proxy state", pod.Name, pod.Namespace)
			}
		}

		return fmt.Errorf("checking the istio-proxy for app %s in namespace %s failed", appName, namespace)
	}, testcontext.GetRetryOpts()...)

}

func ApplicationPodShouldHaveIstioProxyInRequiredVersion(ctx context.Context, appName, namespace string) error {
	requiredProxyVersion := strings.Join([]string{controllers.IstioVersion, controllers.IstioImageBase}, "-")

	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	var podList corev1.PodList
	return retry.Do(func() error {
		podListOpts := &client.ListOptions{
			Namespace: namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{
				"app": appName,
			}),
		}
		err := getPodList(ctx, k8sClient, &podList, podListOpts)

		if err != nil {
			return err
		}

		if len(podList.Items) == 0 {
			return fmt.Errorf("no pods found for app %s in namespace %s", appName, namespace)
		}

		hasProxyInVersion := false
		for _, pod := range podList.Items {
			for _, container := range pod.Spec.Containers {
				if container.Name != "istio-proxy" {
					continue
				}
				deployedVersion, err := getVersionFromImageName(container.Image)
				if err != nil {
					return err
				}

				if deployedVersion == requiredProxyVersion {
					hasProxyInVersion = true
				}
			}
		}

		if !hasProxyInVersion {
			return fmt.Errorf("after upgrade proxy does not match required version")
		}

		return nil
	}, testcontext.GetRetryOpts()...)

}

// CreateHttpbinApplication creates a deployment and a service for the httpbin application
func CreateHttpbinApplication(ctx context.Context, appName, namespace string) (context.Context, error) {
	return CreateHttpbinApplicationWithServicePort(ctx, appName, namespace, 8000)
}

// CreateHttpbinApplication creates a deployment and a service with the given http port for the httpbin application
func CreateHttpbinApplicationWithServicePort(ctx context.Context, appName, namespace string, port int) (context.Context, error) {
	ctx, err := CreateApplicationDeployment(ctx, appName, httpbinImage, namespace)
	if err != nil {
		return ctx, err
	}

	ctx, err = CreateServiceWithPort(ctx, appName, namespace, port, 80)
	if err != nil {
		return ctx, err
	}

	return ctx, err
}

func CreateExtAuthzApplication(ctx context.Context, appName, namespace string) (context.Context, error) {
	c := corev1.Container{
		Name:            appName,
		Image:           extAuthzImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Ports: []corev1.ContainerPort{
			{ContainerPort: 8000},
		},
	}

	ctx, err := CreateServiceWithPort(ctx, appName, namespace, 8000, 8000)
	if err != nil {
		return ctx, err
	}

	ctx, err = CreateDeployment(ctx, appName, namespace, c)
	if err != nil {
		return ctx, err
	}

	return ctx, err
}

func CreateServiceWithPort(ctx context.Context, appName, namespace string, port, targetPort int) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": appName,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       int32(port),
					TargetPort: intstr.IntOrString{IntVal: int32(targetPort)},
				},
			},
		},
	}

	err = retry.Do(func() error {
		err := k8sClient.Create(context.TODO(), &svc)
		if err != nil {
			return err
		}
		ctx = testcontext.AddCreatedTestObjectInContext(ctx, &svc)
		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func getPodList(ctx context.Context, k8sClient client.Client, podList *corev1.PodList, opts *client.ListOptions) error {
	err := k8sClient.List(ctx, podList, opts)
	if err != nil {
		return err
	}
	return nil

}

func getVersionFromImageName(image string) (string, error) {
	noVersion := ""
	matches := reference.ReferenceRegexp.FindStringSubmatch(image)
	if matches == nil || len(matches) < 3 {
		return noVersion, fmt.Errorf("unable to parse container image reference: %s", image)
	}
	version, err := semver.NewVersion(matches[2])
	if err != nil {
		return noVersion, err
	}
	return version.String(), nil
}
