package v1alpha1

import (
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"
	operatorv1alpha1 "istio.io/api/operator/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/pkg/config/mesh"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/types/known/structpb"
	"istio.io/api/mesh/v1alpha1"
	"istio.io/istio/pkg/util/protomarshal"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Merge Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("merge-api-suite", report)
})

var _ = Describe("Merge", func() {

	It("Should update numTrustedProxies on IstioOperator from 1 to 5", func() {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &v1alpha1.Topology{NumTrustedProxies: 1}
		meshConfig := convert(m)

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}

		numProxies := 5
		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(5)))
	})

	It("Should set numTrustedProxies on IstioOperator to 5 when no GatewayTopology is configured", func() {
		// given
		m := mesh.DefaultMeshConfig()
		meshConfig := convert(m)

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}

		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("Should set numTrustedProxies on IstioOperator to 5 when IstioOperator has nil spec", func() {
		// given
		iop := istioOperator.IstioOperator{
			Spec: nil,
		}

		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("Should set numTrustedProxies on IstioOperator to 5 when IstioOperator has nil mesh config", func() {
		// given
		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: nil,
			},
		}

		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("Should change nothing if config is empty", func() {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &v1alpha1.Topology{NumTrustedProxies: 1}
		meshConfig := convert(m)

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}

		istioCR := Istio{Spec: IstioSpec{}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(1)))
	})
	It("Should set numTrustedProxies on IstioOperator to 5 when there is no defaultConfig in meshConfig", func() {
		// given
		m := &v1alpha1.MeshConfig{
			EnableTracing: true,
		}
		meshConfig := convert(m)

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}
		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies := out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue()
		Expect(numTrustedProxies).To(Equal(float64(5)))
	})

	Context("Pilot", func() {
		Context("When Istio CR has 500m configured for CPU limits", func() {
			It("Should set CPU limits to 500m in IOP", func() {
				//given
				iop := istioOperator.IstioOperator{
					Spec: &operatorv1alpha1.IstioOperatorSpec{},
				}
				cpuLimit := "500m"

				istioCR := Istio{Spec: IstioSpec{Components: &Components{
					Pilot: &IstioComponent{K8s: &KubernetesResourcesConfig{
						Resources: &Resources{
							Limits: &ResourceClaims{
								Cpu: &cpuLimit,
							},
						},
					}},
				}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCpuLimit := out.Spec.Components.Pilot.K8S.Resources.Limits["cpu"]
				Expect(iopCpuLimit).To(Equal(cpuLimit))
			})
		})

		Context("When Istio CR has 500m configured for CPU requests", func() {
			It("Should set CPU requests to 500m in IOP", func() {
				//given
				iop := istioOperator.IstioOperator{
					Spec: &operatorv1alpha1.IstioOperatorSpec{},
				}
				cpuLimit := "500m"

				istioCR := Istio{Spec: IstioSpec{Components: &Components{
					Pilot: &IstioComponent{K8s: &KubernetesResourcesConfig{
						Resources: &Resources{
							Requests: &ResourceClaims{
								Cpu: &cpuLimit,
							},
						},
					}},
				}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCpuLimit := out.Spec.Components.Pilot.K8S.Resources.Requests["cpu"]
				Expect(iopCpuLimit).To(Equal(cpuLimit))
			})
		})
	})
	Context("IngressGateway", func() {
		Context("When Istio CR has 500m configured for CPU and 500Mi for memory limits", func() {
			It("Should set CPU limits to 500m and 500Mi for memory in IOP", func() {
				//given
				iop := istioOperator.IstioOperator{
					Spec: &operatorv1alpha1.IstioOperatorSpec{},
				}
				cpuLimit := "500m"
				memoryLimit := "500Mi"

				istioCR := Istio{Spec: IstioSpec{Components: &Components{
					IngressGateway: &IstioComponent{
						K8s: &KubernetesResourcesConfig{
							Resources: &Resources{
								Limits: &ResourceClaims{
									Cpu:    &cpuLimit,
									Memory: &memoryLimit,
								},
							},
						},
					}}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCpuLimit := out.Spec.Components.IngressGateways[0].K8S.Resources.Limits["cpu"]
				Expect(iopCpuLimit).To(Equal(cpuLimit))

				iopMemoryLimit := out.Spec.Components.IngressGateways[0].K8S.Resources.Limits["memory"]
				Expect(iopMemoryLimit).To(Equal(iopMemoryLimit))
			})
		})

		Context("When Istio CR has 500m configured for CPU and 500Mi for memory requests", func() {
			It("Should set CPU requests to 500m and 500Mi for memory in IOP", func() {
				//given
				iop := istioOperator.IstioOperator{
					Spec: &operatorv1alpha1.IstioOperatorSpec{},
				}
				cpuRequests := "500m"
				memoryRequests := "500Mi"

				istioCR := Istio{Spec: IstioSpec{Components: &Components{
					IngressGateway: &IstioComponent{K8s: &KubernetesResourcesConfig{
						Resources: &Resources{
							Requests: &ResourceClaims{
								Cpu:    &cpuRequests,
								Memory: &memoryRequests,
							},
						},
					},
					}}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCpuRequests := out.Spec.Components.IngressGateways[0].K8S.Resources.Requests["cpu"]
				Expect(iopCpuRequests).To(Equal(cpuRequests))

				iopMemoryRequests := out.Spec.Components.IngressGateways[0].K8S.Resources.Requests["memory"]
				Expect(iopMemoryRequests).To(Equal(memoryRequests))
			})
		})
	})

	Context("Strategy", func() {
		It("Should update RollingUpdate when it is present in Istio CR", func() {
			//given
			iop := istioOperator.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{},
			}

			maxUnavailable := intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "50%",
			}

			maxSurge := intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 5,
			}

			istioCR := Istio{Spec: IstioSpec{Components: &Components{
				IngressGateway: &IstioComponent{K8s: &KubernetesResourcesConfig{
					Strategy: &Strategy{
						RollingUpdate: &RollingUpdate{
							MaxUnavailable: &maxUnavailable,
							MaxSurge:       &maxSurge,
						},
					},
				},
				}}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			unavailable := out.Spec.Components.IngressGateways[0].K8S.Strategy.RollingUpdate.MaxUnavailable
			Expect(unavailable.StrVal.GetValue()).To(Equal(maxUnavailable.StrVal))

			surge := out.Spec.Components.IngressGateways[0].K8S.Strategy.RollingUpdate.MaxSurge
			Expect(surge.IntVal.GetValue()).To(Equal(maxSurge.IntVal))
		})
	})

	Context("HPASpec", func() {
		It("Should update HPASpec when it is present in Istio CR", func() {
			//given
			iop := istioOperator.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{},
			}
			maxReplicas := int32(5)
			minReplicas := int32(4)

			istioCR := Istio{Spec: IstioSpec{Components: &Components{
				IngressGateway: &IstioComponent{K8s: &KubernetesResourcesConfig{
					HPASpec: &HPASpec{
						MaxReplicas: &maxReplicas,
						MinReplicas: &minReplicas,
					},
				},
				}}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			replicas := out.Spec.Components.IngressGateways[0].K8S.HpaSpec.MaxReplicas
			Expect(replicas).To(Equal(maxReplicas))

			replicas = out.Spec.Components.IngressGateways[0].K8S.HpaSpec.MinReplicas
			Expect(replicas).To(Equal(minReplicas))
		})
	})

	Context("CNI", func() {
		Context("Affinity", func() {
			Context("PodAffinity", func() {
				It("Should update CNI affinity when it is present in Istio CR", func() {
					//given
					iop := istioOperator.IstioOperator{
						Spec: &operatorv1alpha1.IstioOperatorSpec{},
					}

					istioCR := Istio{Spec: IstioSpec{Components: &Components{
						Cni: &CniComponent{K8S: &CniK8sConfig{
							Affinity: &v1.Affinity{
								PodAffinity: &v1.PodAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
										{
											LabelSelector: &metav1.LabelSelector{
												MatchExpressions: []metav1.LabelSelectorRequirement{
													{
														Key:      "app-new",
														Operator: "In",
														Values:   Label("istio-cni-node1"),
													},
												},
											},
										},
									},
								},
							},
						}},
					}}}

					// when
					out, err := istioCR.MergeInto(iop)

					// then
					Expect(err).ShouldNot(HaveOccurred())

					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Key).To(Equal("app-new"))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Operator).To(BeEquivalentTo("In"))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values[0]).To(BeEquivalentTo("istio-cni-node1"))
				})
			})

			Context("PodAntiAffinity", func() {
				It("Should update CNI PodAntiAffinity when it is present in Istio CR", func() {
					//given
					iop := istioOperator.IstioOperator{
						Spec: &operatorv1alpha1.IstioOperatorSpec{},
					}

					istioCR := Istio{Spec: IstioSpec{Components: &Components{
						Cni: &CniComponent{K8S: &CniK8sConfig{
							Affinity: &v1.Affinity{
								PodAntiAffinity: &v1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
										{
											LabelSelector: &metav1.LabelSelector{
												MatchExpressions: []metav1.LabelSelectorRequirement{
													{
														Key:      "app-new",
														Operator: "In",
														Values:   Label("istio-cni-node1"),
													},
												},
											},
										},
									},
								},
							},
						}},
					}}}

					// when
					out, err := istioCR.MergeInto(iop)

					// then
					Expect(err).ShouldNot(HaveOccurred())

					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Key).To(Equal("app-new"))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Operator).To(BeEquivalentTo("In"))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values[0]).To(BeEquivalentTo("istio-cni-node1"))
				})
			})

			Context("NodeAffinity", func() {
				It("Should update CNI NodeAffinity when it is present in Istio CR", func() {
					//given
					iop := istioOperator.IstioOperator{
						Spec: &operatorv1alpha1.IstioOperatorSpec{},
					}

					istioCR := Istio{Spec: IstioSpec{Components: &Components{
						Cni: &CniComponent{K8S: &CniK8sConfig{
							Affinity: &v1.Affinity{
								NodeAffinity: &v1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
										NodeSelectorTerms: []v1.NodeSelectorTerm{
											{
												MatchExpressions: []v1.NodeSelectorRequirement{
													{
														Key:      "app-new",
														Operator: "In",
														Values:   Label("istio-cni-node1"),
													},
												},
											},
										},
									},
								},
							},
						}},
					}}}

					// when
					out, err := istioCR.MergeInto(iop)

					// then
					Expect(err).ShouldNot(HaveOccurred())

					Expect(out.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key).To(Equal("app-new"))
					Expect(out.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Operator).To(BeEquivalentTo("In"))
					Expect(out.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0]).To(BeEquivalentTo("istio-cni-node1"))
				})
			})
		})

		Context("Resources", func() {
			It("Should update CNI resources when those are present in Istio CR", func() {
				//given
				iop := istioOperator.IstioOperator{
					Spec: &operatorv1alpha1.IstioOperatorSpec{},
				}
				cpuRequests := "500m"
				memoryRequests := "500Mi"

				istioCR := Istio{Spec: IstioSpec{Components: &Components{
					Cni: &CniComponent{K8S: &CniK8sConfig{
						Resources: &Resources{
							Requests: &ResourceClaims{
								Cpu:    &cpuRequests,
								Memory: &memoryRequests,
							},
						},
					}},
				}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCpuRequests := out.Spec.Components.Cni.K8S.Resources.Requests["cpu"]
				Expect(iopCpuRequests).To(Equal(cpuRequests))

				iopMemoryRequests := out.Spec.Components.Cni.K8S.Resources.Requests["memory"]
				Expect(iopMemoryRequests).To(Equal(memoryRequests))
			})
		})
	})

	Context("Proxy", func() {
		It("Should update Proxy resources configuration if they are present in Istio CR", func() {
			//given
			iop := istioOperator.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{},
			}

			cpuRequests := "500m"
			memoryRequests := "500Mi"

			cpuLimits := "800m"
			memoryLimits := "800Mi"
			istioCR := Istio{Spec: IstioSpec{Components: &Components{
				Proxy: &ProxyComponent{K8S: &ProxyK8sConfig{
					Resources: &Resources{
						Requests: &ResourceClaims{
							Cpu:    &cpuRequests,
							Memory: &memoryRequests,
						},
						Limits: &ResourceClaims{
							Cpu:    &cpuLimits,
							Memory: &memoryLimits,
						},
					},
				}},
			}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			resources := out.Spec.Values.Fields["global"].GetStructValue().Fields["proxy"].GetStructValue().Fields["resources"].GetStructValue()
			Expect(resources.Fields["requests"].GetStructValue().Fields["cpu"].GetStringValue()).To(Equal(cpuRequests))
			Expect(resources.Fields["requests"].GetStructValue().Fields["memory"].GetStringValue()).To(Equal(memoryRequests))

			Expect(resources.Fields["limits"].GetStructValue().Fields["cpu"].GetStringValue()).To(Equal(cpuLimits))
			Expect(resources.Fields["limits"].GetStructValue().Fields["memory"].GetStringValue()).To(Equal(memoryLimits))
		})
	})

})

func convert(a *v1alpha1.MeshConfig) *structpb.Struct {

	mMap, err := protomarshal.ToJSONMap(a)
	Expect(err).ShouldNot(HaveOccurred())

	mStruct, err := structpb.NewStruct(mMap)
	Expect(err).ShouldNot(HaveOccurred())

	return mStruct
}
