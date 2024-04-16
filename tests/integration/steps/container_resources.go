package steps

import (
	"context"
	"fmt"
	"github.com/avast/retry-go"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/pkg/errors"
	"gopkg.in/inf.v0"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

type resourceStruct struct {
	Cpu    resource.Quantity
	Memory resource.Quantity
}

func DeploymentHasPodWithContainerResourcesSetToCpuAndMemory(ctx context.Context, depName, depNamespace, container, resourceType, cpu, memory string) (context.Context, error) {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return ctx, err
	}

	err = retry.Do(func() error {
		pods, err := getPodsControlledByDeployment(k8sClient, depNamespace, depName)
		if err != nil {
			return err
		}

		if len(pods) == 0 {
			return errors.Errorf("no pods found for deployment %s/%s", depNamespace, depName)
		}

		pod := pods[0]
		resources, err := getContainerResources(pod, container, resourceType)
		if err != nil {
			return err
		}

		if err := assertResources(resources, cpu, memory); err != nil {
			return errors.Wrap(err, fmt.Sprintf("assert %s resources of container %s in pod %s/%s", resourceType, container, pod.Namespace, pod.Name))
		}

		return nil
	}, testcontext.GetRetryOpts()...)

	return ctx, err
}

func getPodsControlledByDeployment(k8sClient client.Client, depNamespace string, depName string) ([]v1.Pod, error) {
	var dep appsv1.Deployment
	if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: depNamespace, Name: depName}, &dep); err != nil {
		return nil, err
	}

	var rsList appsv1.ReplicaSetList
	if err := k8sClient.List(context.Background(), &rsList, client.MatchingLabels(dep.Spec.Selector.MatchLabels)); err != nil {
		return nil, err
	}

	if len(rsList.Items) == 0 {
		return nil, fmt.Errorf("no replica sets found for deployment %s/%s", depNamespace, depName)
	}

	podLabelFilter := client.MatchingLabels(rsList.Items[0].Spec.Selector.MatchLabels)

	podList := &v1.PodList{}
	if err := k8sClient.List(context.Background(), podList, podLabelFilter); err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func getContainerResources(pod v1.Pod, container, resourceType string) (resourceStruct, error) {
	for _, c := range pod.Spec.Containers {
		if c.Name == container {
			switch resourceType {
			case "limits":
				return resourceStruct{
					Cpu:    *c.Resources.Limits.Cpu(),
					Memory: *c.Resources.Limits.Memory(),
				}, nil
			case "requests":
				return resourceStruct{
					Cpu:    *c.Resources.Requests.Cpu(),
					Memory: *c.Resources.Requests.Memory(),
				}, nil
			default:
				return resourceStruct{}, fmt.Errorf("resource type %s is not supported", resourceType)
			}
		}
	}

	return resourceStruct{}, fmt.Errorf("container istio-proxy not found in pod %s/%s", pod.Namespace, pod.Name)
}

func assertResources(actualResources resourceStruct, expectedCpu, expectedMemory string) error {

	cpuMilli, err := strconv.Atoi(strings.TrimSuffix(expectedCpu, "m"))
	if err != nil {
		return err
	}

	memMilli, err := strconv.Atoi(strings.TrimSuffix(expectedMemory, "Mi"))
	if err != nil {
		return err
	}

	if resource.NewDecimalQuantity(*inf.NewDec(int64(cpuMilli), inf.Scale(resource.Milli)), resource.DecimalSI).Equal(actualResources.Cpu) {
		return fmt.Errorf("cpu wasn't expected; expected=%v got=%v", resource.NewScaledQuantity(int64(cpuMilli), resource.Milli), actualResources.Cpu)
	}

	if resource.NewDecimalQuantity(*inf.NewDec(int64(memMilli), inf.Scale(resource.Milli)), resource.DecimalSI).Equal(actualResources.Memory) {
		return fmt.Errorf("memory wasn't expected; expected=%v got=%v", resource.NewScaledQuantity(int64(memMilli), resource.Milli), actualResources.Memory)
	}

	return nil
}
