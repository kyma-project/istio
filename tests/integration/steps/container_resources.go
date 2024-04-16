package steps

import (
	"context"
	"fmt"
	"github.com/kyma-project/istio/operator/tests/integration/testcontext"
	"github.com/pkg/errors"
	"gopkg.in/inf.v0"
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

func DeploymentHasPodWithContainerResourcesSetToCpuAndMemory(ctx context.Context, depName, depNamespace, container, resourceType, cpu, memory string) error {
	k8sClient, err := testcontext.GetK8sClientFromContext(ctx)
	if err != nil {
		return err
	}

	podList := &v1.PodList{}
	opts := []client.ListOption{
		client.InNamespace(depNamespace),
		client.MatchingLabels{"app": depName},
	}
	if err := k8sClient.List(context.Background(), podList, opts...); err != nil {
		return err
	}

	if len(podList.Items) == 0 {
		return errors.Errorf("no pods found for deployment %s/%s", depNamespace, depName)
	}

	pod := podList.Items[0]
	resources, err := getContainerResources(pod, container, resourceType)
	if err != nil {
		return err
	}

	if err := assertResources(resources, cpu, memory); err != nil {
		return errors.Wrap(err, fmt.Sprintf("assert %s resources of container %s in pod %s/%s", resourceType, container, pod.Namespace, pod.Name))
	}
	return nil
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
