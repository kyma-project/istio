package resources

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/inf.v0"
	"k8s.io/apimachinery/pkg/api/resource"
)

type ResourceStruct struct {
	Cpu    resource.Quantity
	Memory resource.Quantity
}

func NewResourceStruct(cpu, memory resource.Quantity) ResourceStruct {
	return ResourceStruct{
		Cpu:    cpu,
		Memory: memory,
	}
}

func AssertResources(actualResources ResourceStruct, expectedCpu, expectedMemory string) error {

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