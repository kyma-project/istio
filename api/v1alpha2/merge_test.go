package v1alpha2_test

import (
	"encoding/json"
	"testing"

	istiov1alpha2 "github.com/kyma-project/istio/operator/api/v1alpha2"

	"github.com/onsi/ginkgo/v2/types"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
	meshv1alpha1 "istio.io/api/mesh/v1alpha1"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis"
	"istio.io/istio/operator/pkg/values"
	"istio.io/istio/pkg/config/mesh"
	"istio.io/istio/pkg/util/protomarshal"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"

	"github.com/kyma-project/istio/operator/internal/tests"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo tests are generally written without a direct package reference
	. "github.com/onsi/gomega"    //nolint:revive // Gomega asserts are generally written without a direct package reference
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Merge Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("merge-api-suite", report)
})

const (
	HeadersToUpstreamOnAllow        = "headersToUpstreamOnAllow"
	HeadersToDownstreamOnAllow      = "headersToDownstreamOnAllow"
	HeadersToDownstreamOnDeny       = "headersToDownstreamOnDeny"
	IncludeAdditionalHeadersInCheck = "includeAdditionalHeadersInCheck"
	IncludeRequestHeadersInCheck    = "includeRequestHeadersInCheck"
)

var _ = Describe("Merge", func() {
	Context("Authorizations", func() {
		It("should set authorizer with no headers setup", func() {
			// given
			m := mesh.DefaultMeshConfig()
			meshConfigRaw := convert(m)

			iop := iopv1alpha1.IstioOperator{
				Spec: iopv1alpha1.IstioOperatorSpec{
					MeshConfig: meshConfigRaw,
				},
			}

			provName := "test-authorizer"

			authorizer := istiov1alpha2.Authorizer{
				Name:    provName,
				Service: "xauth",
				Port:    1337,
			}
			istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{Authorizers: []*istiov1alpha2.Authorizer{
				&authorizer,
			}}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
			Expect(err).ShouldNot(HaveOccurred())

			extensionProvidersInt, exists := meshConfig.GetPath("extensionProviders")
			Expect(exists).To(BeTrue())

			extensionProviders := extensionProvidersInt.([]interface{})

			var foundAuthorizer bool
			for _, extensionProviderInt := range extensionProviders {
				extensionProvider, ok := extensionProviderInt.(map[string]interface{})
				Expect(ok).To(BeTrue())

				if extensionProvider["name"] == provName {
					extensionProviderMap, mapErr := values.MapFromObject(extensionProvider)
					Expect(mapErr).ShouldNot(HaveOccurred())

					authProvider, okGetPath := extensionProviderMap.GetPathMap("envoyExtAuthzHttp")
					Expect(okGetPath).To(BeTrue())

					Expect(authProvider).ShouldNot(BeNil())
					Expect(authProvider["port"]).To(BeEquivalentTo(1337))
					Expect(authProvider["service"]).To(Equal("xauth"))

					foundAuthorizer = true
					break
				}
			}

			Expect(foundAuthorizer).To(BeTrue(), "Could not find the authorizer by the name")
		})

		It("should set multiple authorizers", func() {
			// given
			m := mesh.DefaultMeshConfig()
			meshConfigRaw := convert(m)
			iop := iopv1alpha1.IstioOperator{
				Spec: iopv1alpha1.IstioOperatorSpec{
					MeshConfig: meshConfigRaw,
				},
			}

			provName1 := "test-authorizer-1"
			provName2 := "test-authorizer-2"

			authorizer1 := istiov1alpha2.Authorizer{
				Name:    provName1,
				Service: "xauth1",
				Port:    1337,
			}

			authorizer2 := istiov1alpha2.Authorizer{
				Name:    provName2,
				Service: "xauth2",
				Port:    2337,
			}

			istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{Authorizers: []*istiov1alpha2.Authorizer{
				&authorizer1,
				&authorizer2,
			}}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())
			meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
			Expect(err).ShouldNot(HaveOccurred())
			extensionProvidersInt, exists := meshConfig.GetPath("extensionProviders")
			Expect(exists).To(BeTrue())
			extensionProviders := extensionProvidersInt.([]interface{})
			var foundAuthorizer1, foundAuthorizer2 bool
			for _, extensionProviderInt := range extensionProviders {
				extensionProvider, ok := extensionProviderInt.(map[string]interface{})
				Expect(ok).To(BeTrue())

				switch extensionProvider["name"] {
				case provName1:
					extensionProviderMap, errMap := values.MapFromObject(extensionProvider)
					Expect(errMap).ShouldNot(HaveOccurred())

					authProvider, okGetPath := extensionProviderMap.GetPathMap("envoyExtAuthzHttp")
					Expect(okGetPath).To(BeTrue())

					Expect(authProvider).ShouldNot(BeNil())
					Expect(authProvider["port"]).To(BeEquivalentTo(1337))
					Expect(authProvider["service"]).To(Equal("xauth1"))

					foundAuthorizer1 = true
				case provName2:
					extensionProviderMap, errMap := values.MapFromObject(extensionProvider)
					Expect(errMap).ShouldNot(HaveOccurred())

					authProvider, okGetPath := extensionProviderMap.GetPathMap("envoyExtAuthzHttp")
					Expect(okGetPath).To(BeTrue())

					Expect(authProvider).ShouldNot(BeNil())
					Expect(authProvider["port"]).To(BeEquivalentTo(2337))
					Expect(authProvider["service"]).To(Equal("xauth2"))

					foundAuthorizer2 = true
				}
			}
			Expect(foundAuthorizer1).To(BeTrue(), "Could not find the first authorizer by the name")
			Expect(foundAuthorizer2).To(BeTrue(), "Could not find the second authorizer by the name")
		})

		It("should set headers for authorizer", func() {
			// given
			m := mesh.DefaultMeshConfig()
			meshConfigRaw := convert(m)

			iop := iopv1alpha1.IstioOperator{
				Spec: iopv1alpha1.IstioOperatorSpec{
					MeshConfig: meshConfigRaw,
				},
			}

			provName := "test-authorizer"

			inCheckInclude := []string{
				"authorization",
				"test",
			}

			inCheckAdd := map[string]string{
				"a": "a",
				"b": "b",
			}

			toUpstreamOnAllow := []string{
				"c",
				"d",
			}

			toDownstreamOnAllow := []string{
				"da",
				"db",
			}

			toDownstreamOnDeny := []string{
				"dc",
				"dd",
			}

			authorizer := istiov1alpha2.Authorizer{
				Name:    provName,
				Service: "xauth",
				Port:    1337,
				Headers: &istiov1alpha2.Headers{
					InCheck: &istiov1alpha2.InCheck{
						Include: inCheckInclude,
						Add:     inCheckAdd,
					},
					ToUpstream: &istiov1alpha2.ToUpstream{
						OnAllow: toUpstreamOnAllow,
					},
					ToDownstream: &istiov1alpha2.ToDownstream{
						OnAllow: toDownstreamOnAllow,
						OnDeny:  toDownstreamOnDeny,
					},
				},
			}

			istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{Authorizers: []*istiov1alpha2.Authorizer{
				&authorizer,
			}}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
			Expect(err).ShouldNot(HaveOccurred())

			extensionProvidersInt, exists := meshConfig.GetPath("extensionProviders")
			Expect(exists).To(BeTrue())

			extensionProviders := extensionProvidersInt.([]interface{})

			var foundAuthorizer bool
			for _, extensionProviderInt := range extensionProviders {
				extensionProvider, ok := extensionProviderInt.(map[string]interface{})
				Expect(ok).To(BeTrue())

				if extensionProvider["name"] == provName {
					extensionProviderMap, errMap := values.MapFromObject(extensionProvider)
					Expect(errMap).ShouldNot(HaveOccurred())

					authProvider, okGetPath := extensionProviderMap.GetPathMap("envoyExtAuthzHttp")
					Expect(okGetPath).To(BeTrue())

					Expect(authProvider).ShouldNot(BeNil())
					Expect(authProvider["port"]).To(BeEquivalentTo(1337))
					Expect(authProvider["service"]).To(Equal("xauth"))

					Expect(authProvider[HeadersToUpstreamOnAllow]).To(ConsistOf(toUpstreamOnAllow))
					Expect(authProvider[HeadersToDownstreamOnAllow]).To(ConsistOf(toDownstreamOnAllow))
					Expect(authProvider[HeadersToDownstreamOnDeny]).To(ConsistOf(toDownstreamOnDeny))

					Expect(authProvider[IncludeAdditionalHeadersInCheck]).To(HaveKeyWithValue("a", "a"))
					Expect(authProvider[IncludeAdditionalHeadersInCheck]).To(HaveKeyWithValue("b", "b"))
					Expect(authProvider[IncludeRequestHeadersInCheck]).To(ConsistOf(inCheckInclude))

					foundAuthorizer = true
					break
				}
			}

			Expect(foundAuthorizer).To(BeTrue(), "Could not find the authorizer by the name")
		})
	})

	It("should update numTrustedProxies on IstioOperator from 1 to 5", func() {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &meshv1alpha1.Topology{NumTrustedProxies: 1}
		meshConfigRaw := convert(m)

		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfigRaw,
			},
		}

		numProxies := 5
		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies, exists := meshConfig.GetPath("defaultConfig.gatewayTopology.numTrustedProxies")
		Expect(exists).To(BeTrue())
		Expect(numTrustedProxies).To(Equal(float64(5)))
	})

	It("should set numTrustedProxies on IstioOperator to 5 when no GatewayTopology is configured", func() {
		// given
		m := mesh.DefaultMeshConfig()
		meshConfigRaw := convert(m)

		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfigRaw,
			},
		}

		numProxies := 5

		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies, exists := meshConfig.GetPath("defaultConfig.gatewayTopology.numTrustedProxies")
		Expect(exists).To(BeTrue())
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("should set numTrustedProxies on IstioOperator to 5 when IstioOperator has nil spec", func() {
		// given
		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{},
		}

		numProxies := 5

		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies, exists := meshConfig.GetPath("defaultConfig.gatewayTopology.numTrustedProxies")
		Expect(exists).To(BeTrue())
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("should set numTrustedProxies on IstioOperator to 5 when IstioOperator has nil mesh config", func() {
		// given
		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{
				MeshConfig: nil,
			},
		}

		numProxies := 5

		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies, exists := meshConfig.GetPath("defaultConfig.gatewayTopology.numTrustedProxies")
		Expect(exists).To(BeTrue())
		Expect(numTrustedProxies).To(Equal(float64(numProxies)))
	})

	It("should change nothing if config is empty", func() {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &meshv1alpha1.Topology{NumTrustedProxies: 1}
		meshConfigRaw := convert(m)

		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfigRaw,
			},
		}

		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies, exists := meshConfig.GetPath("defaultConfig.gatewayTopology.numTrustedProxies")
		Expect(exists).To(BeTrue())
		Expect(numTrustedProxies).To(Equal(float64(1)))
	})
	It("should set numTrustedProxies on IstioOperator to 5 when there is no defaultConfig in meshConfig", func() {
		// given
		m := &meshv1alpha1.MeshConfig{
			EnableTracing: true,
		}
		meshConfigRaw := convert(m)

		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfigRaw,
			},
		}
		numProxies := 5

		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
		Expect(err).ShouldNot(HaveOccurred())

		numTrustedProxies, exists := meshConfig.GetPath("defaultConfig.gatewayTopology.numTrustedProxies")
		Expect(exists).To(BeTrue())
		Expect(numTrustedProxies).To(Equal(float64(5)))
	})

	It("should set prometheusMerge on IstioOperator Telemetry Metrics to true when meshConfig has a enablePrometheusMerge with true", func() {
		// given
		m := &meshv1alpha1.MeshConfig{
			EnablePrometheusMerge: wrapperspb.Bool(false),
		}
		meshConfigRaw := convert(m)

		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfigRaw,
			},
		}
		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Config: istiov1alpha2.Config{
			Telemetry: istiov1alpha2.Telemetry{
				Metrics: istiov1alpha2.Metrics{
					PrometheusMerge: true,
				},
			},
		}}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		meshConfig, err := values.MapFromObject(out.Spec.MeshConfig)
		Expect(err).ShouldNot(HaveOccurred())

		enabledPrometheusMerge, exists := meshConfig.GetPath("enablePrometheusMerge")
		Expect(exists).To(BeTrue())
		Expect(enabledPrometheusMerge).To(BeTrue())

	})

	It("should create IngressGateway overlay with externalTrafficPolicy set to Local", func() {
		// given

		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{},
		}
		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{
			Config: istiov1alpha2.Config{
				GatewayExternalTrafficPolicy: ptr.To("Local"),
			},
		}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		externalTrafficPolicy := out.Spec.Components.IngressGateways[0].Kubernetes.Overlays[0].Patches[0].Value.(*structpb.Value).GetStringValue()
		Expect(externalTrafficPolicy).To(Equal("Local"))
	})

	It("should create IngressGateway overlay with externalTrafficPolicy set to Cluster", func() {
		// given

		iop := iopv1alpha1.IstioOperator{
			Spec: iopv1alpha1.IstioOperatorSpec{},
		}
		istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{
			Config: istiov1alpha2.Config{
				GatewayExternalTrafficPolicy: ptr.To("Cluster"),
			},
		}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		externalTrafficPolicy := out.Spec.Components.IngressGateways[0].Kubernetes.Overlays[0].Patches[0].Value.(*structpb.Value).GetStringValue()
		Expect(externalTrafficPolicy).To(Equal("Cluster"))

	})

	Context("Pilot", func() {
		Context("When Istio CR has 500m configured for CPU limits", func() {
			It("should set CPU limits to 500m in IOP", func() {
				// given
				iop := iopv1alpha1.IstioOperator{
					Spec: iopv1alpha1.IstioOperatorSpec{},
				}
				cpuLimit := "500m"

				istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
					Pilot: &istiov1alpha2.IstioComponent{K8s: &istiov1alpha2.KubernetesResourcesConfig{
						Resources: &istiov1alpha2.Resources{
							Limits: &istiov1alpha2.ResourceClaims{
								CPU: &cpuLimit,
							},
						},
					}},
				}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCPULimit := out.Spec.Components.Pilot.Kubernetes.Resources.Limits["cpu"]
				Expect(iopCPULimit.String()).To(Equal(cpuLimit))
			})
		})

		Context("When Istio CR has 500m configured for CPU requests", func() {
			It("should set CPU requests to 500m in IOP", func() {
				// given
				iop := iopv1alpha1.IstioOperator{
					Spec: iopv1alpha1.IstioOperatorSpec{},
				}
				cpuLimit := "500m"

				istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
					Pilot: &istiov1alpha2.IstioComponent{K8s: &istiov1alpha2.KubernetesResourcesConfig{
						Resources: &istiov1alpha2.Resources{
							Requests: &istiov1alpha2.ResourceClaims{
								CPU: &cpuLimit,
							},
						},
					}},
				}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCPULimit := out.Spec.Components.Pilot.Kubernetes.Resources.Requests["cpu"]
				Expect(iopCPULimit.String()).To(Equal(cpuLimit))
			})
		})
	})

	Context("IngressGateway", func() {
		Context("When Istio CR has 500m configured for CPU and 500Mi for memory limits", func() {
			It("should set CPU limits to 500m and 500Mi for memory in IOP", func() {
				// given
				iop := iopv1alpha1.IstioOperator{
					Spec: iopv1alpha1.IstioOperatorSpec{},
				}
				cpuLimit := "500m"
				memoryLimit := "500Mi"

				istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
					IngressGateway: &istiov1alpha2.IstioComponent{
						K8s: &istiov1alpha2.KubernetesResourcesConfig{
							Resources: &istiov1alpha2.Resources{
								Limits: &istiov1alpha2.ResourceClaims{
									CPU:    &cpuLimit,
									Memory: &memoryLimit,
								},
							},
						},
					}}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCPULimit := out.Spec.Components.IngressGateways[0].Kubernetes.Resources.Limits["cpu"]
				Expect(iopCPULimit.String()).To(Equal(cpuLimit))

				iopMemoryLimit := out.Spec.Components.IngressGateways[0].Kubernetes.Resources.Limits["memory"]
				Expect(iopMemoryLimit.String()).To(Equal(memoryLimit))
			})
		})

		Context("When Istio CR has 500m configured for CPU and 500Mi for memory requests", func() {
			It("should set CPU requests to 500m and 500Mi for memory in IOP", func() {
				// given
				iop := iopv1alpha1.IstioOperator{
					Spec: iopv1alpha1.IstioOperatorSpec{},
				}
				cpuRequests := "500m"
				memoryRequests := "500Mi"

				istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
					IngressGateway: &istiov1alpha2.IstioComponent{K8s: &istiov1alpha2.KubernetesResourcesConfig{
						Resources: &istiov1alpha2.Resources{
							Requests: &istiov1alpha2.ResourceClaims{
								CPU:    &cpuRequests,
								Memory: &memoryRequests,
							},
						},
					},
					}}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCPURequests := out.Spec.Components.IngressGateways[0].Kubernetes.Resources.Requests["cpu"]
				Expect(iopCPURequests.String()).To(Equal(cpuRequests))

				iopMemoryRequests := out.Spec.Components.IngressGateways[0].Kubernetes.Resources.Requests["memory"]
				Expect(iopMemoryRequests.String()).To(Equal(memoryRequests))
			})
		})
	})

	Context("EgressGateway", func() {
		Context("When Istio CR has 500m configured for CPU and 500Mi for memory limits", func() {
			It("should set CPU limits to 500m and 500Mi for memory in IOP", func() {
				// given
				iop := iopv1alpha1.IstioOperator{
					Spec: iopv1alpha1.IstioOperatorSpec{},
				}
				cpuLimit := "500m"
				memoryLimit := "500Mi"
				enabled := true

				istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
					EgressGateway: &istiov1alpha2.EgressGateway{
						Enabled: &enabled,
						K8s: &istiov1alpha2.KubernetesResourcesConfig{
							Resources: &istiov1alpha2.Resources{
								Limits: &istiov1alpha2.ResourceClaims{
									CPU:    &cpuLimit,
									Memory: &memoryLimit,
								},
							},
						},
					}}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCPULimit := ptr.To(out.Spec.Components.EgressGateways[0].Kubernetes.Resources.Limits["cpu"])
				Expect(iopCPULimit.String()).To(Equal(cpuLimit))

				iopMemoryLimit := ptr.To(out.Spec.Components.EgressGateways[0].Kubernetes.Resources.Limits["memory"])
				Expect(iopMemoryLimit.String()).To(Equal(memoryLimit))

				iopEnabled := out.Spec.Components.EgressGateways[0].Enabled.GetValueOrFalse()
				Expect(iopEnabled).To(Equal(enabled))
			})
		})

		Context("When Istio CR has 500m configured for CPU and 500Mi for memory requests", func() {
			It("should set CPU requests to 500m and 500Mi for memory in IOP", func() {
				// given
				iop := iopv1alpha1.IstioOperator{
					Spec: iopv1alpha1.IstioOperatorSpec{},
				}
				cpuRequests := "500m"
				memoryRequests := "500Mi"
				enabled := true

				istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
					EgressGateway: &istiov1alpha2.EgressGateway{
						Enabled: &enabled,
						K8s: &istiov1alpha2.KubernetesResourcesConfig{
							Resources: &istiov1alpha2.Resources{
								Requests: &istiov1alpha2.ResourceClaims{
									CPU:    &cpuRequests,
									Memory: &memoryRequests,
								},
							},
						},
					}}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCPURequests := ptr.To(out.Spec.Components.EgressGateways[0].Kubernetes.Resources.Requests["cpu"])
				Expect(iopCPURequests.String()).To(Equal(cpuRequests))

				iopMemoryRequests := ptr.To(out.Spec.Components.EgressGateways[0].Kubernetes.Resources.Requests["memory"])
				Expect(iopMemoryRequests.String()).To(Equal(memoryRequests))

				iopEnabled := out.Spec.Components.EgressGateways[0].Enabled.GetValueOrFalse()
				Expect(iopEnabled).To(Equal(enabled))
			})
		})
	})

	Context("Strategy", func() {
		It("should update RollingUpdate when it is present in Istio CR", func() {
			// given
			iop := iopv1alpha1.IstioOperator{
				Spec: iopv1alpha1.IstioOperatorSpec{},
			}

			maxUnavailable := intstr.IntOrString{
				Type:   intstr.String,
				StrVal: "50%",
			}

			maxSurge := intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 5,
			}

			istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
				IngressGateway: &istiov1alpha2.IstioComponent{K8s: &istiov1alpha2.KubernetesResourcesConfig{
					Strategy: &istiov1alpha2.Strategy{
						RollingUpdate: &istiov1alpha2.RollingUpdate{
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

			unavailable := out.Spec.Components.IngressGateways[0].Kubernetes.Strategy.RollingUpdate.MaxUnavailable
			Expect(unavailable.StrVal).To(Equal(maxUnavailable.StrVal))

			surge := out.Spec.Components.IngressGateways[0].Kubernetes.Strategy.RollingUpdate.MaxSurge
			Expect(surge.IntVal).To(Equal(maxSurge.IntVal))
		})
	})

	Context("HPASpec", func() {
		It("should update HPASpec when it is present in Istio CR", func() {
			// given
			iop := iopv1alpha1.IstioOperator{
				Spec: iopv1alpha1.IstioOperatorSpec{},
			}
			maxReplicas := int32(5)
			minReplicas := int32(4)

			istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
				IngressGateway: &istiov1alpha2.IstioComponent{K8s: &istiov1alpha2.KubernetesResourcesConfig{
					HPASpec: &istiov1alpha2.HPASpec{
						MaxReplicas: &maxReplicas,
						MinReplicas: &minReplicas,
					},
				},
				}}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			replicas := out.Spec.Components.IngressGateways[0].Kubernetes.HpaSpec.MaxReplicas
			Expect(replicas).To(Equal(maxReplicas))

			replicas = *out.Spec.Components.IngressGateways[0].Kubernetes.HpaSpec.MinReplicas
			Expect(replicas).To(Equal(minReplicas))
		})
	})

	Context("CNI", func() {
		Context("Affinity", func() {
			Context("PodAffinity", func() {
				It("should update CNI affinity when it is present in Istio CR", func() {
					// given
					iop := iopv1alpha1.IstioOperator{
						Spec: iopv1alpha1.IstioOperatorSpec{},
					}

					istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
						Cni: &istiov1alpha2.CniComponent{K8S: &istiov1alpha2.CniK8sConfig{
							Affinity: &corev1.Affinity{
								PodAffinity: &corev1.PodAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
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

					Expect(out.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
					Expect(out.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions).To(HaveLen(1))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Key,
					).To(Equal("app-new"))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Operator,
					).To(BeEquivalentTo("In"))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values,
					).To(HaveLen(1))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values[0],
					).To(BeEquivalentTo("istio-cni-node1"))
				})
			})

			Context("PodAntiAffinity", func() {
				It("should update CNI PodAntiAffinity when it is present in Istio CR", func() {
					// given
					iop := iopv1alpha1.IstioOperator{
						Spec: iopv1alpha1.IstioOperatorSpec{},
					}

					istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
						Cni: &istiov1alpha2.CniComponent{K8S: &istiov1alpha2.CniK8sConfig{
							Affinity: &corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
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

					Expect(out.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution).To(HaveLen(1))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions,
					).To(HaveLen(1))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Key,
					).To(Equal("app-new"))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Operator,
					).To(BeEquivalentTo("In"))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values,
					).To(HaveLen(1))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution[0].LabelSelector.MatchExpressions[0].Values[0],
					).To(BeEquivalentTo("istio-cni-node1"))
				})
			})

			Context("NodeAffinity", func() {
				It("should update CNI NodeAffinity when it is present in Istio CR", func() {
					// given
					iop := iopv1alpha1.IstioOperator{
						Spec: iopv1alpha1.IstioOperatorSpec{},
					}

					istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
						Cni: &istiov1alpha2.CniComponent{K8S: &istiov1alpha2.CniK8sConfig{
							Affinity: &corev1.Affinity{
								NodeAffinity: &corev1.NodeAffinity{
									RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
										NodeSelectorTerms: []corev1.NodeSelectorTerm{
											{
												MatchExpressions: []corev1.NodeSelectorRequirement{
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

					Expect(out.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms).To(HaveLen(1))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions,
					).To(HaveLen(1))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Key,
					).To(Equal("app-new"))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Operator,
					).To(BeEquivalentTo("In"))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values,
					).To(HaveLen(1))
					Expect(
						out.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0].MatchExpressions[0].Values[0],
					).To(BeEquivalentTo("istio-cni-node1"))
				})
			})
		})

		Context("Resources", func() {
			It("should update CNI resources when those are present in Istio CR", func() {
				// given
				iop := iopv1alpha1.IstioOperator{
					Spec: iopv1alpha1.IstioOperatorSpec{},
				}
				cpuRequests := "500m"
				memoryRequests := "500Mi"

				istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
					Cni: &istiov1alpha2.CniComponent{K8S: &istiov1alpha2.CniK8sConfig{
						Resources: &istiov1alpha2.Resources{
							Requests: &istiov1alpha2.ResourceClaims{
								CPU:    &cpuRequests,
								Memory: &memoryRequests,
							},
						},
					}},
				}}}

				// when
				out, err := istioCR.MergeInto(iop)

				// then
				Expect(err).ShouldNot(HaveOccurred())

				iopCPURequests := out.Spec.Components.Cni.Kubernetes.Resources.Requests["cpu"]
				Expect(iopCPURequests.String()).To(Equal(cpuRequests))

				iopMemoryRequests := out.Spec.Components.Cni.Kubernetes.Resources.Requests["memory"]
				Expect(iopMemoryRequests.String()).To(Equal(memoryRequests))
			})
		})
	})

	Context("Proxy", func() {
		It("should update Proxy resources configuration if they are present in Istio CR", func() {
			// given
			iop := iopv1alpha1.IstioOperator{
				Spec: iopv1alpha1.IstioOperatorSpec{},
			}

			cpuRequests := "500m"
			memoryRequests := "500Mi"

			cpuLimits := "800m"
			memoryLimits := "800Mi"
			istioCR := istiov1alpha2.Istio{Spec: istiov1alpha2.IstioSpec{Components: &istiov1alpha2.Components{
				Proxy: &istiov1alpha2.ProxyComponent{K8S: &istiov1alpha2.ProxyK8sConfig{
					Resources: &istiov1alpha2.Resources{
						Requests: &istiov1alpha2.ResourceClaims{
							CPU:    &cpuRequests,
							Memory: &memoryRequests,
						},
						Limits: &istiov1alpha2.ResourceClaims{
							CPU:    &cpuLimits,
							Memory: &memoryLimits,
						},
					},
				}},
			}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			valuesMap, err := values.MapFromObject(out.Spec.Values)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(values.TryGetPathAs[string](valuesMap, "global.proxy.resources.requests.cpu")).To(Equal(cpuRequests))
			Expect(values.TryGetPathAs[string](valuesMap, "global.proxy.resources.requests.memory")).To(Equal(memoryRequests))
			Expect(values.TryGetPathAs[string](valuesMap, "global.proxy.resources.limits.cpu")).To(Equal(cpuLimits))
			Expect(values.TryGetPathAs[string](valuesMap, "global.proxy.resources.limits.memory")).To(Equal(memoryLimits))
		})
	})

})

func convert(a *meshv1alpha1.MeshConfig) json.RawMessage {
	jsonConfig, err := protomarshal.Marshal(a)
	Expect(err).ShouldNot(HaveOccurred())

	return jsonConfig
}
