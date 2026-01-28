package resourceassert

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/inf.v0"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/e2e-framework/klient/k8s/resources"
	"sigs.k8s.io/e2e-framework/klient/wait"
)

// AssertIstioProxyResourcesForPod asserts that the istio-proxy container in the pod has the expected resource requests and limits.
// For init containers, it looks for the istio-proxy in InitContainers.
// For regular containers, it looks for the istio-proxy in Containers.
func AssertIstioProxyResourcesForPod(t *testing.T, pod v1.Pod, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory string) {
	// Try to find istio-proxy in Containers first
	for _, container := range pod.Spec.Containers {
		if container.Name == "istio-proxy" {
			AssertContainerResources(t, container, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory)
			return
		}
	}

	// If not found in Containers, look in InitContainers
	for _, container := range pod.Spec.InitContainers {
		if container.Name == "istio-proxy" {
			AssertContainerResources(t, container, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory)
			return
		}
	}

	require.Fail(t, "istio-proxy container not found in pod", "Pod: %s", pod.Name)
}

// AssertIstioProxyResourcesEventually waits for pods matching the label selector to have the expected istio-proxy resources.
// This is useful when resources are updated, and you need to wait for pods to be restarted with the new configuration.
func AssertIstioProxyResourcesEventually(
	t *testing.T,
	c *resources.Resources,
	labelSelector string,
	expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory string,
) {
	t.Helper()

	err := wait.For(func(ctx context.Context) (bool, error) {
		podList := &v1.PodList{}
		err := c.List(ctx, podList, resources.WithLabelSelector(labelSelector))
		if err != nil {
			return false, err
		}

		if len(podList.Items) == 0 {
			return false, fmt.Errorf("no pods found for label selector %s", labelSelector)
		}

		for _, pod := range podList.Items {
			if pod.Status.Phase != v1.PodRunning {
				return false, nil
			}

			found := false
			for _, container := range append(pod.Spec.Containers, pod.Spec.InitContainers...) {
				if container.Name == "istio-proxy" {
					found = true
					if !checkContainerResources(container, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory) {
						return false, nil
					}
					break
				}
			}

			if !found {
				return false, fmt.Errorf("istio-proxy container not found in pod %s", pod.Name)
			}
		}

		return true, nil
	}, wait.WithTimeout(1*time.Minute), wait.WithInterval(5*time.Second), wait.WithContext(t.Context()))

	require.NoError(t, err, "Failed to verify proxy resources for label selector %s", labelSelector)
}

// checkContainerResources checks if a container has the expected resource values
func checkContainerResources(container v1.Container, cpuRequest, memRequest, cpuLimit, memLimit string) bool {
	expectedCPURequest := resource.MustParse(cpuRequest)
	expectedMemRequest := resource.MustParse(memRequest)
	expectedCPULimit := resource.MustParse(cpuLimit)
	expectedMemLimit := resource.MustParse(memLimit)

	actualCPURequest := container.Resources.Requests[v1.ResourceCPU]
	actualMemRequest := container.Resources.Requests[v1.ResourceMemory]
	actualCPULimit := container.Resources.Limits[v1.ResourceCPU]
	actualMemLimit := container.Resources.Limits[v1.ResourceMemory]

	return actualCPURequest.Equal(expectedCPURequest) &&
		actualMemRequest.Equal(expectedMemRequest) &&
		actualCPULimit.Equal(expectedCPULimit) &&
		actualMemLimit.Equal(expectedMemLimit)
}

// AssertContainerResources asserts that a container has the expected resource requests and limits
func AssertContainerResources(t *testing.T, container v1.Container, expectedRequestCpu, expectedRequestMemory, expectedLimitCpu, expectedLimitMemory string) {
	err := assertResources(resourceStruct{
		Cpu:    *container.Resources.Requests.Cpu(),
		Memory: *container.Resources.Requests.Memory(),
	}, expectedRequestCpu, expectedRequestMemory)
	require.NoError(t, err)

	err = assertResources(resourceStruct{
		Cpu:    *container.Resources.Limits.Cpu(),
		Memory: *container.Resources.Limits.Memory(),
	}, expectedLimitCpu, expectedLimitMemory)
	require.NoError(t, err)
}

type resourceStruct struct {
	Cpu    resource.Quantity
	Memory resource.Quantity
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
