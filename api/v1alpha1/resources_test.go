package v1alpha1_test

import (
	v1alpha1 "github.com/kyma-project/istio/operator/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/pointer"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Resources", func() {

	Context("IsEqual", func() {

		tests := []struct {
			name     string
			r1       v1alpha1.Resources
			r2       v1.ResourceRequirements
			expected bool
		}{
			{
				name: "should be equal when requests and limits are the same",
				r1: v1alpha1.Resources{
					Requests: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("100m"),
						Memory: pointer.String("200Mi"),
					},
					Limits: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("200m"),
						Memory: pointer.String("400Mi"),
					},
				},
				r2: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("200m"),
						v1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
				expected: true,
			},
			{
				name: "should not be equal when requests cpu is different",
				r1: v1alpha1.Resources{
					Requests: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("500m"),
						Memory: pointer.String("200Mi"),
					},
					Limits: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("200m"),
						Memory: pointer.String("400Mi"),
					},
				},
				r2: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("200m"),
						v1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
				expected: false,
			},
			{
				name: "should not be equal when requests memory is different",
				r1: v1alpha1.Resources{
					Requests: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("100m"),
						Memory: pointer.String("800Mi"),
					},
					Limits: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("200m"),
						Memory: pointer.String("400Mi"),
					},
				},
				r2: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("200m"),
						v1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
				expected: false,
			}, {
				name: "should not be equal when limits cpu is different",
				r1: v1alpha1.Resources{
					Requests: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("100m"),
						Memory: pointer.String("200Mi"),
					},
					Limits: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("500m"),
						Memory: pointer.String("400Mi"),
					},
				},
				r2: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("200m"),
						v1.ResourceMemory: resource.MustParse("400Mi"),
					},
				},
				expected: false,
			}, {
				name: "should not be equal when limits memory is different",
				r1: v1alpha1.Resources{
					Requests: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("100m"),
						Memory: pointer.String("200Mi"),
					},
					Limits: &v1alpha1.ResourceClaims{
						Cpu:    pointer.String("200m"),
						Memory: pointer.String("400Mi"),
					},
				},
				r2: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						v1.ResourceCPU:    resource.MustParse("100m"),
						v1.ResourceMemory: resource.MustParse("200Mi"),
					},
					Limits: v1.ResourceList{
						v1.ResourceCPU: resource.MustParse("200m"),
					},
				},
				expected: false,
			},
		}

		for _, tt := range tests {
			It(tt.name, func() {
				result := tt.r1.IsEqual(tt.r2)
				Expect(result).To(Equal(tt.expected))
			})
		}
	})

})
