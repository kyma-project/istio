package v1alpha2

import (
	"encoding/json"
	"github.com/kyma-project/istio/operator/internal/tests"
	"github.com/onsi/ginkgo/v2/types"
	meshv1alpha1 "istio.io/api/mesh/v1alpha1"
	operatorv1alpha1 "istio.io/api/operator/v1alpha1"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/pkg/config/mesh"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"regexp"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"google.golang.org/protobuf/types/known/structpb"
	"istio.io/istio/pkg/util/protomarshal"
)

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Merge Suite")
}

var _ = ReportAfterSuite("custom reporter", func(report types.Report) {
	tests.GenerateGinkgoJunitReport("merge-api-suite", report)
})

func toSnakeCase(str string) string {
	matchFirstCap := regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCap := regexp.MustCompile("([a-z0-9])([A-Z])")

	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// Struct_pb uses different conventions for json tags (snake case) and unmarshalling (camel case).
// Because of this difference, json tags need to be translated to snake_case before unmarshalling;
// otherwise those fields would become nil.
func jsonTagsToSnakeCase(camelCaseMarshaledJson []byte) string {
	jsonString := string(camelCaseMarshaledJson)
	tagMatch := regexp.MustCompile(`"[^ ]*" *:`)
	return tagMatch.ReplaceAllStringFunc(jsonString, toSnakeCase)
}

var _ = Describe("Merge", func() {
	Context("Authorizations", func() {
		It("should set authorizer with no headers setup", func() {
			// given
			m := mesh.DefaultMeshConfig()
			meshConfig := convert(m)

			iop := iopv1alpha1.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{
					MeshConfig: meshConfig,
				},
			}

			provName := "test-authorizer"

			authorizer := Authorizer{
				Name:    provName,
				Service: "xauth",
				Port:    1337,
			}
			istioCR := Istio{Spec: IstioSpec{Config: Config{Authorizers: []*Authorizer{
				&authorizer,
			}}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			extensionProviders := out.Spec.MeshConfig.Fields["extensionProviders"].GetListValue().GetValues()
			var foundAuthorizer bool
			for _, extensionProvider := range extensionProviders {
				if extensionProvider.GetStructValue().Fields["name"].GetStringValue() == provName {
					jsonAuthProvider, err := extensionProvider.GetStructValue().Fields["envoyExtAuthzHttp"].MarshalJSON()
					jsonAuthProvider = []byte(jsonTagsToSnakeCase(jsonAuthProvider))
					Expect(err).ToNot(HaveOccurred())

					var authProvider meshv1alpha1.MeshConfig_ExtensionProvider_EnvoyExternalAuthorizationHttpProvider
					err = json.Unmarshal(jsonAuthProvider, &authProvider)
					Expect(err).ToNot(HaveOccurred())

					Expect(authProvider.Port).To(BeEquivalentTo(1337))
					Expect(authProvider.Service).To(Equal("xauth"))

					foundAuthorizer = true
					break
				}
			}

			Expect(foundAuthorizer).To(BeTrue(), "Could not find the authorizer by the name")
		})

		It("should set headers for authorizer", func() {
			// given
			m := mesh.DefaultMeshConfig()
			meshConfig := convert(m)

			iop := iopv1alpha1.IstioOperator{
				Spec: &operatorv1alpha1.IstioOperatorSpec{
					MeshConfig: meshConfig,
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

			authorizer := Authorizer{
				Name:    provName,
				Service: "xauth",
				Port:    1337,
				Headers: &Headers{
					InCheck: &InCheck{
						Include: inCheckInclude,
						Add:     inCheckAdd,
					},
					ToUpstream: &ToUpstream{
						OnAllow: toUpstreamOnAllow,
					},
					ToDownstream: &ToDownstream{
						OnAllow: toDownstreamOnAllow,
						OnDeny:  toDownstreamOnDeny,
					},
				},
			}

			istioCR := Istio{Spec: IstioSpec{Config: Config{Authorizers: []*Authorizer{
				&authorizer,
			}}}}

			// when
			out, err := istioCR.MergeInto(iop)

			// then
			Expect(err).ShouldNot(HaveOccurred())

			extensionProviders := out.Spec.MeshConfig.Fields["extensionProviders"].GetListValue().GetValues()
			var foundAuthorizer bool
			for _, extensionProvider := range extensionProviders {
				if extensionProvider.GetStructValue().Fields["name"].GetStringValue() == provName {
					jsonAuthProvider, err := extensionProvider.GetStructValue().Fields["envoyExtAuthzHttp"].MarshalJSON()
					jsonAuthProvider = []byte(jsonTagsToSnakeCase(jsonAuthProvider))
					Expect(err).ToNot(HaveOccurred())

					var authProvider meshv1alpha1.MeshConfig_ExtensionProvider_EnvoyExternalAuthorizationHttpProvider
					err = json.Unmarshal(jsonAuthProvider, &authProvider)
					Expect(err).ToNot(HaveOccurred())

					Expect(authProvider.Port).To(BeEquivalentTo(1337))
					Expect(authProvider.Service).To(Equal("xauth"))

					Expect(authProvider.HeadersToUpstreamOnAllow).To(ConsistOf(toUpstreamOnAllow))
					Expect(authProvider.HeadersToDownstreamOnAllow).To(ConsistOf(toDownstreamOnAllow))
					Expect(authProvider.HeadersToDownstreamOnDeny).To(ConsistOf(toDownstreamOnDeny))

					Expect(authProvider.IncludeAdditionalHeadersInCheck).To(HaveKeyWithValue("a", "a"))
					Expect(authProvider.IncludeAdditionalHeadersInCheck).To(HaveKeyWithValue("b", "b"))
					Expect(authProvider.IncludeRequestHeadersInCheck).To(ConsistOf(inCheckInclude))

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
		meshConfig := convert(m)

		iop := iopv1alpha1.IstioOperator{
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

	It("should set numTrustedProxies on IstioOperator to 5 when no GatewayTopology is configured", func() {
		// given
		m := mesh.DefaultMeshConfig()
		meshConfig := convert(m)

		iop := iopv1alpha1.IstioOperator{
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

	It("should set numTrustedProxies on IstioOperator to 5 when IstioOperator has nil spec", func() {
		// given
		iop := iopv1alpha1.IstioOperator{
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

	It("should set numTrustedProxies on IstioOperator to 5 when IstioOperator has nil mesh config", func() {
		// given
		iop := iopv1alpha1.IstioOperator{
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

	It("should change nothing if config is empty", func() {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &meshv1alpha1.Topology{NumTrustedProxies: 1}
		meshConfig := convert(m)

		iop := iopv1alpha1.IstioOperator{
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
	It("should set numTrustedProxies on IstioOperator to 5 when there is no defaultConfig in meshConfig", func() {
		// given
		m := &meshv1alpha1.MeshConfig{
			EnableTracing: true,
		}
		meshConfig := convert(m)

		iop := iopv1alpha1.IstioOperator{
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

	It("should create IngressGateway overlay with externalTrafficPolicy set to Local", func() {
		// given

		iop := iopv1alpha1.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{},
		}
		istioCR := Istio{Spec: IstioSpec{
			Config: Config{
				GatewayExternalTrafficPolicy: ptr.To("Local"),
			},
		}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		externalTrafficPolicy := out.Spec.Components.IngressGateways[0].K8S.Overlays[0].Patches[0].Value.GetStringValue()
		Expect(externalTrafficPolicy).To(Equal("Local"))
	})

	It("should create IngressGateway overlay with externalTrafficPolicy set to Cluster", func() {
		// given

		iop := iopv1alpha1.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{},
		}
		istioCR := Istio{Spec: IstioSpec{
			Config: Config{
				GatewayExternalTrafficPolicy: ptr.To("Cluster"),
			},
		}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		Expect(err).ShouldNot(HaveOccurred())

		externalTrafficPolicy := out.Spec.Components.IngressGateways[0].K8S.Overlays[0].Patches[0].Value.GetStringValue()
		Expect(externalTrafficPolicy).To(Equal("Cluster"))

	})

	Context("Pilot", func() {
		Context("When Istio CR has 500m configured for CPU limits", func() {
			It("should set CPU limits to 500m in IOP", func() {
				//given
				iop := iopv1alpha1.IstioOperator{
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
			It("should set CPU requests to 500m in IOP", func() {
				//given
				iop := iopv1alpha1.IstioOperator{
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
		Context("Istio CR annotation to disable external name alias feature", func() {
			It("should set env variable to true when there was no annotation", func() {
				//given
				iop := iopv1alpha1.IstioOperator{
					Spec: &operatorv1alpha1.IstioOperatorSpec{},
				}
				istioCR := Istio{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{},
					},
					Spec: IstioSpec{},
				}

				// when
				out, err := istioCR.MergeInto(iop)

				var env *operatorv1alpha1.EnvVar
				//then
				Expect(err).ShouldNot(HaveOccurred())
				for _, v := range out.Spec.Components.Pilot.K8S.Env {
					if v.Name == "ENABLE_EXTERNAL_NAME_ALIAS" {
						env = v
					}
				}
				Expect(env).ToNot(BeNil())
				Expect(env.Value).To(Equal("true"))
			})
			It("should set env variable to true when the annotation value is false", func() {
				//given
				iop := iopv1alpha1.IstioOperator{
					Spec: &operatorv1alpha1.IstioOperatorSpec{},
				}
				istioCR := Istio{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"istio-operator.kyma-project.io/disable-external-name-alias": "false",
						},
					},
					Spec: IstioSpec{},
				}

				// when
				out, err := istioCR.MergeInto(iop)

				//then
				Expect(err).ShouldNot(HaveOccurred())
				var env *operatorv1alpha1.EnvVar
				for _, v := range out.Spec.Components.Pilot.K8S.Env {
					if v.Name == "ENABLE_EXTERNAL_NAME_ALIAS" {
						env = v
					}
				}
				Expect(env).ToNot(BeNil())
				Expect(env.Value).To(Equal("true"))
			})
			It("should set env variable to false when the annotation value is true", func() {
				//given
				iop := iopv1alpha1.IstioOperator{
					Spec: &operatorv1alpha1.IstioOperatorSpec{},
				}
				istioCR := Istio{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"istio-operator.kyma-project.io/disable-external-name-alias": "true",
						},
					},
					Spec: IstioSpec{},
				}

				// when
				out, err := istioCR.MergeInto(iop)

				var env *operatorv1alpha1.EnvVar
				//then
				Expect(err).ShouldNot(HaveOccurred())
				for _, v := range out.Spec.Components.Pilot.K8S.Env {
					if v.Name == "ENABLE_EXTERNAL_NAME_ALIAS" {
						env = v
					}
				}
				Expect(env).ToNot(BeNil())
				Expect(env.Value).To(Equal("false"))
			})
		})
	})

	Context("IngressGateway", func() {
		Context("When Istio CR has 500m configured for CPU and 500Mi for memory limits", func() {
			It("should set CPU limits to 500m and 500Mi for memory in IOP", func() {
				//given
				iop := iopv1alpha1.IstioOperator{
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
			It("should set CPU requests to 500m and 500Mi for memory in IOP", func() {
				//given
				iop := iopv1alpha1.IstioOperator{
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
		It("should update RollingUpdate when it is present in Istio CR", func() {
			//given
			iop := iopv1alpha1.IstioOperator{
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
		It("should update HPASpec when it is present in Istio CR", func() {
			//given
			iop := iopv1alpha1.IstioOperator{
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
				It("should update CNI affinity when it is present in Istio CR", func() {
					//given
					iop := iopv1alpha1.IstioOperator{
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
				It("should update CNI PodAntiAffinity when it is present in Istio CR", func() {
					//given
					iop := iopv1alpha1.IstioOperator{
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
				It("should update CNI NodeAffinity when it is present in Istio CR", func() {
					//given
					iop := iopv1alpha1.IstioOperator{
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
			It("should update CNI resources when those are present in Istio CR", func() {
				//given
				iop := iopv1alpha1.IstioOperator{
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
		It("should update Proxy resources configuration if they are present in Istio CR", func() {
			//given
			iop := iopv1alpha1.IstioOperator{
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

func convert(a *meshv1alpha1.MeshConfig) *structpb.Struct {

	mMap, err := protomarshal.ToJSONMap(a)
	Expect(err).ShouldNot(HaveOccurred())

	mStruct, err := structpb.NewStruct(mMap)
	Expect(err).ShouldNot(HaveOccurred())

	return mStruct
}
