package resourceassert

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/inf.v0"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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
