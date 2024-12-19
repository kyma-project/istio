package v1alpha2

import (
	"encoding/json"
	"istio.io/istio/operator/pkg/values"
	"istio.io/istio/pkg/util/protomarshal"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"

	"google.golang.org/protobuf/types/known/structpb"
	meshv1alpha1 "istio.io/api/mesh/v1alpha1"
	iopv1alpha1 "istio.io/istio/operator/pkg/apis"
)

func (i *Istio) MergeInto(op iopv1alpha1.IstioOperator) (iopv1alpha1.IstioOperator, error) {
	mergedConfigOp, err := i.mergeConfig(op)
	if err != nil {
		return op, err
	}

	mergedResourcesOp, err := i.mergeResources(mergedConfigOp)
	if err != nil {
		return op, err
	}

	if i.Spec.CompatibilityMode {
		compatibleIop, err := setCompatibilityMode(mergedResourcesOp)
		if err != nil {
			return op, err
		}
		return compatibleIop, nil
	}

	return mergedResourcesOp, nil
}

type meshConfigBuilder struct {
	c values.Map
}

func newMeshConfigBuilder(op iopv1alpha1.IstioOperator) (*meshConfigBuilder, error) {
	if op.Spec.MeshConfig == nil {
		return &meshConfigBuilder{c: make(values.Map)}, nil
	}
	c, err := values.MapFromObject(op.Spec.MeshConfig)
	if err != nil {
		return nil, err
	}

	return &meshConfigBuilder{c: c}, nil
}

func (m *meshConfigBuilder) BuildNumTrustedProxies(numTrustedProxies *int) *meshConfigBuilder {
	if numTrustedProxies == nil {
		return m
	}

	err := m.c.SetPath("defaultConfig.gatewayTopology.numTrustedProxies", numTrustedProxies)
	if err != nil {
		return nil
	}

	return m
}

func (m *meshConfigBuilder) BuildPrometheusMergeConfig(prometheusMerge bool) *meshConfigBuilder {
	err := m.c.SetPath("enablePrometheusMerge", prometheusMerge)
	if err != nil {
		return nil
	}
	return m
}

func (m *meshConfigBuilder) AddProxyMetadata(key, value string) (*meshConfigBuilder, error) {
	err := m.c.SetPath("defaultConfig.proxyMetadata."+key, value)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *meshConfigBuilder) Build() json.RawMessage {
	return json.RawMessage(m.c.JSON())
}

func (m *meshConfigBuilder) BuildExternalAuthorizerConfiguration(authorizers []*Authorizer) *meshConfigBuilder {
	for _, authorizer := range authorizers {
		if authorizer != nil {
			var authorizationProvider meshv1alpha1.MeshConfig_ExtensionProvider
			authorizationProvider.Name = authorizer.Name
			var envoyXAuthProvider meshv1alpha1.MeshConfig_ExtensionProvider_EnvoyExtAuthzHttp
			envoyXAuthProvider.EnvoyExtAuthzHttp = &meshv1alpha1.MeshConfig_ExtensionProvider_EnvoyExternalAuthorizationHttpProvider{
				Service: authorizer.Service,
				Port:    authorizer.Port,
			}

			headers := authorizer.Headers
			if headers != nil {
				if headers.InCheck != nil {
					include := headers.InCheck.Include
					if include != nil {
						envoyXAuthProvider.EnvoyExtAuthzHttp.IncludeRequestHeadersInCheck = append(envoyXAuthProvider.EnvoyExtAuthzHttp.IncludeRequestHeadersInCheck, include...)
					}

					add := headers.InCheck.Add
					if add != nil {
						envoyXAuthProvider.EnvoyExtAuthzHttp.IncludeAdditionalHeadersInCheck = make(map[string]string)
						for k, v := range add {
							envoyXAuthProvider.EnvoyExtAuthzHttp.IncludeAdditionalHeadersInCheck[k] = v
						}
					}
				}

				if headers.ToUpstream != nil {
					onAllow := headers.ToUpstream.OnAllow
					if onAllow != nil {
						envoyXAuthProvider.EnvoyExtAuthzHttp.HeadersToUpstreamOnAllow = append(envoyXAuthProvider.EnvoyExtAuthzHttp.HeadersToUpstreamOnAllow, onAllow...)
					}
				}

				if headers.ToDownstream != nil {
					onAllow := headers.ToDownstream.OnAllow
					if onAllow != nil {
						envoyXAuthProvider.EnvoyExtAuthzHttp.HeadersToDownstreamOnAllow = append(envoyXAuthProvider.EnvoyExtAuthzHttp.HeadersToDownstreamOnAllow, onAllow...)
					}

					onDeny := headers.ToDownstream.OnDeny
					if onDeny != nil {
						envoyXAuthProvider.EnvoyExtAuthzHttp.HeadersToDownstreamOnDeny = append(envoyXAuthProvider.EnvoyExtAuthzHttp.HeadersToDownstreamOnDeny, onDeny...)
					}
				}
			}

			authorizationProvider.Provider = &envoyXAuthProvider
			extensionProviders := values.TryGetPathAs[[]interface{}](m.c, "extensionProviders")
			marshaledProvider, err := protomarshal.Marshal(&authorizationProvider)
			if err != nil {
				return nil
			}
			var providerMap map[string]interface{}
			err = json.Unmarshal(marshaledProvider, &providerMap)
			if err != nil {
				return nil
			}

			extensionProviders = append(extensionProviders, providerMap)
			err = m.c.SetPath("extensionProviders", &extensionProviders)
			if err != nil {
				return nil
			}
		}
	}

	return m
}

func (i *Istio) mergeConfig(op iopv1alpha1.IstioOperator) (iopv1alpha1.IstioOperator, error) {

	mcb, err := newMeshConfigBuilder(op)
	if err != nil {
		return op, err
	}

	newMeshConfig := mcb.
		BuildNumTrustedProxies(i.Spec.Config.NumTrustedProxies).
		BuildExternalAuthorizerConfiguration(i.Spec.Config.Authorizers).
		BuildPrometheusMergeConfig(i.Spec.Config.EnablePrometheusMerge).
		Build()

	op.Spec.MeshConfig = newMeshConfig

	op = applyGatewayExternalTrafficPolicy(op, i)

	return op, nil
}

func applyGatewayExternalTrafficPolicy(op iopv1alpha1.IstioOperator, i *Istio) iopv1alpha1.IstioOperator {
	if i.Spec.Config.GatewayExternalTrafficPolicy != nil {
		if op.Spec.Components == nil {
			op.Spec.Components = &iopv1alpha1.IstioComponentSpec{}
		}
		if len(op.Spec.Components.IngressGateways) == 0 {
			op.Spec.Components.IngressGateways = append(op.Spec.Components.IngressGateways, iopv1alpha1.GatewayComponentSpec{})
		}
		if op.Spec.Components.IngressGateways[0].Kubernetes == nil {
			op.Spec.Components.IngressGateways[0].Kubernetes = &iopv1alpha1.KubernetesResources{}
		}

		const kind = "Service"
		const version = "v1"
		const istioIngressGateway = "istio-ingressgateway"
		const path = "spec.externalTrafficPolicy"

		op.Spec.Components.IngressGateways[0].Kubernetes.Overlays = append(op.Spec.Components.IngressGateways[0].Kubernetes.Overlays, iopv1alpha1.KubernetesOverlay{
			ApiVersion: version,
			Kind:       kind,
			Name:       istioIngressGateway,
			Patches: []iopv1alpha1.Patch{
				{
					Path:  path,
					Value: structpb.NewStringValue(*i.Spec.Config.GatewayExternalTrafficPolicy),
				},
			},
		})
	}
	return op
}

func (i *Istio) mergeResources(op iopv1alpha1.IstioOperator) (iopv1alpha1.IstioOperator, error) {
	if i.Spec.Components == nil {
		return op, nil
	}

	if i.Spec.Components.IngressGateway != nil {
		if op.Spec.Components == nil {
			op.Spec.Components = &iopv1alpha1.IstioComponentSpec{}
		}
		if len(op.Spec.Components.IngressGateways) == 0 {
			op.Spec.Components.IngressGateways = append(op.Spec.Components.IngressGateways, iopv1alpha1.GatewayComponentSpec{})
		}
		if op.Spec.Components.IngressGateways[0].Kubernetes == nil {
			op.Spec.Components.IngressGateways[0].Kubernetes = &iopv1alpha1.KubernetesResources{}
		}
		if i.Spec.Components.IngressGateway.K8s != nil {
			err := mergeK8sConfig(op.Spec.Components.IngressGateways[0].Kubernetes, *i.Spec.Components.IngressGateway.K8s)
			if err != nil {
				return op, err
			}
		}
	}

	if i.Spec.Components.EgressGateway != nil {
		if op.Spec.Components == nil {
			op.Spec.Components = &iopv1alpha1.IstioComponentSpec{}
		}
		if len(op.Spec.Components.EgressGateways) == 0 {
			op.Spec.Components.EgressGateways = append(op.Spec.Components.EgressGateways, iopv1alpha1.GatewayComponentSpec{})
		}
		if op.Spec.Components.EgressGateways[0].Kubernetes == nil {
			op.Spec.Components.EgressGateways[0].Kubernetes = &iopv1alpha1.KubernetesResources{}
		}
		if i.Spec.Components.EgressGateway.K8s != nil {
			err := mergeK8sConfig(op.Spec.Components.EgressGateways[0].Kubernetes, *i.Spec.Components.EgressGateway.K8s)
			if err != nil {
				return op, err
			}
		}
		if i.Spec.Components.EgressGateway.Enabled != nil {
			if op.Spec.Components.EgressGateways[0].Enabled == nil {
				op.Spec.Components.EgressGateways[0].Enabled = &iopv1alpha1.BoolValue{}
			}
			boolValue := iopv1alpha1.BoolValue{}
			// This terrible if statement is necessary, because Istio decided to use a custom type for booleans,
			//that stores bool as a private field, and does not have a constructor/setter, only an unmarshal method.
			if *i.Spec.Components.EgressGateway.Enabled {
				err := boolValue.UnmarshalJSON([]byte("true"))
				if err != nil {
					return op, err
				}
				op.Spec.Components.EgressGateways[0].Enabled = &boolValue
			} else {
				err := boolValue.UnmarshalJSON([]byte("false"))
				if err != nil {
					return op, err
				}
				op.Spec.Components.EgressGateways[0].Enabled = &boolValue
			}
		}
	}

	if i.Spec.Components.Pilot != nil {
		if op.Spec.Components == nil {
			op.Spec.Components = &iopv1alpha1.IstioComponentSpec{}
		}
		if op.Spec.Components.Pilot == nil {
			op.Spec.Components.Pilot = &iopv1alpha1.ComponentSpec{}
		}
		if op.Spec.Components.Pilot.Kubernetes == nil {
			op.Spec.Components.Pilot.Kubernetes = &iopv1alpha1.KubernetesResources{}
		}
		if i.Spec.Components.Pilot.K8s != nil {
			err := mergeK8sConfig(op.Spec.Components.Pilot.Kubernetes, *i.Spec.Components.Pilot.K8s)
			if err != nil {
				return op, err
			}
		}
	}

	valuesMap, err := values.MapFromObject(op.Spec.Values)
	if err != nil {
		return op, err
	}

	if valuesMap == nil {
		valuesMap = make(values.Map)
	}

	if i.Spec.Components.Proxy != nil && i.Spec.Components.Proxy.K8S != nil && i.Spec.Components.Proxy.K8S.Resources != nil {
		if i.Spec.Components.Proxy.K8S.Resources.Limits != nil {
			if i.Spec.Components.Proxy.K8S.Resources.Limits.Cpu != nil {
				err = valuesMap.SetPath("global.proxy.resources.limits.cpu", *i.Spec.Components.Proxy.K8S.Resources.Limits.Cpu)
				if err != nil {
					return iopv1alpha1.IstioOperator{}, err
				}
			}
			if i.Spec.Components.Proxy.K8S.Resources.Limits.Memory != nil {
				err = valuesMap.SetPath("global.proxy.resources.limits.memory", *i.Spec.Components.Proxy.K8S.Resources.Limits.Memory)
				if err != nil {
					return iopv1alpha1.IstioOperator{}, err
				}
			}
		}

		if i.Spec.Components.Proxy.K8S.Resources.Requests != nil {
			if i.Spec.Components.Proxy.K8S.Resources.Requests != nil {
				if i.Spec.Components.Proxy.K8S.Resources.Requests.Cpu != nil {
					err = valuesMap.SetPath("global.proxy.resources.requests.cpu", *i.Spec.Components.Proxy.K8S.Resources.Requests.Cpu)
					if err != nil {
						return iopv1alpha1.IstioOperator{}, err
					}
				}

				if i.Spec.Components.Proxy.K8S.Resources.Requests.Memory != nil {
					err = valuesMap.SetPath("global.proxy.resources.requests.memory", *i.Spec.Components.Proxy.K8S.Resources.Requests.Memory)
					if err != nil {
						return iopv1alpha1.IstioOperator{}, err
					}
				}
			}
		}
		op.Spec.Values, err = values.ConvertMap[json.RawMessage](valuesMap)
		if err != nil {
			return op, err
		}
	}

	if i.Spec.Components.Cni != nil {
		if op.Spec.Components == nil {
			op.Spec.Components = &iopv1alpha1.IstioComponentSpec{}
		}

		if op.Spec.Components.Cni == nil {
			op.Spec.Components.Cni = &iopv1alpha1.ComponentSpec{}
		}

		if op.Spec.Components.Cni.Kubernetes == nil {
			op.Spec.Components.Cni.Kubernetes = &iopv1alpha1.KubernetesResources{}
		}

		if op.Spec.Components.Cni.Kubernetes.Affinity == nil {
			op.Spec.Components.Cni.Kubernetes.Affinity = &corev1.Affinity{}
		}

		if i.Spec.Components.Cni.K8S != nil && i.Spec.Components.Cni.K8S.Affinity != nil {
			if op.Spec.Components.Cni.Kubernetes.Affinity == nil {
				op.Spec.Components.Cni.Kubernetes.Affinity = &corev1.Affinity{}
			}
			if i.Spec.Components.Cni.K8S.Affinity.PodAffinity != nil {
				if op.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity == nil {
					op.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity = &corev1.PodAffinity{}
				}
				op.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = i.Spec.Components.Cni.K8S.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution
				op.Spec.Components.Cni.Kubernetes.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = i.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			}

			if i.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity != nil {
				if op.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity == nil {
					op.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
				}
				op.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = i.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
				op.Spec.Components.Cni.Kubernetes.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = i.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			}

			if i.Spec.Components.Cni.K8S.Affinity.NodeAffinity != nil {
				if op.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity == nil {
					op.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity = &corev1.NodeAffinity{}
				}
				op.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = i.Spec.Components.Cni.K8S.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution
				op.Spec.Components.Cni.Kubernetes.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = i.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
			}
		}

		if i.Spec.Components.Cni.K8S.Resources != nil {
			if op.Spec.Components.Cni.Kubernetes.Resources == nil {
				op.Spec.Components.Cni.Kubernetes.Resources = &corev1.ResourceRequirements{}
			}

			if i.Spec.Components.Cni.K8S.Resources.Limits != nil {
				if op.Spec.Components.Cni.Kubernetes.Resources.Limits == nil {
					op.Spec.Components.Cni.Kubernetes.Resources.Limits = make(corev1.ResourceList)
				}

				if i.Spec.Components.Cni.K8S.Resources.Limits.Cpu != nil {
					quantity, err := resource.ParseQuantity(*i.Spec.Components.Cni.K8S.Resources.Limits.Cpu)
					if err != nil {
						return op, err
					}
					op.Spec.Components.Cni.Kubernetes.Resources.Limits[corev1.ResourceCPU] = quantity
				}
				if i.Spec.Components.Cni.K8S.Resources.Limits.Memory != nil {
					quantity, err := resource.ParseQuantity(*i.Spec.Components.Cni.K8S.Resources.Limits.Memory)
					if err != nil {
						return op, err
					}
					op.Spec.Components.Cni.Kubernetes.Resources.Limits[corev1.ResourceMemory] = quantity
				}
			}

			if i.Spec.Components.Cni.K8S.Resources.Requests != nil {
				if op.Spec.Components.Cni.Kubernetes.Resources.Requests == nil {
					op.Spec.Components.Cni.Kubernetes.Resources.Requests = make(corev1.ResourceList)
				}

				if i.Spec.Components.Cni.K8S.Resources.Requests.Cpu != nil {
					quantity, err := resource.ParseQuantity(*i.Spec.Components.Cni.K8S.Resources.Requests.Cpu)
					if err != nil {
						return op, err
					}
					op.Spec.Components.Cni.Kubernetes.Resources.Requests[corev1.ResourceCPU] = quantity
				}
				if i.Spec.Components.Cni.K8S.Resources.Requests.Memory != nil {
					quantity, err := resource.ParseQuantity(*i.Spec.Components.Cni.K8S.Resources.Requests.Memory)
					if err != nil {
						return op, err
					}
					op.Spec.Components.Cni.Kubernetes.Resources.Requests[corev1.ResourceMemory] = quantity
				}
			}
		}
	}

	return op, nil
}

func mergeK8sConfig(base *iopv1alpha1.KubernetesResources, newConfig KubernetesResourcesConfig) error {
	if newConfig.Resources != nil {
		if base.Resources == nil {
			base.Resources = &corev1.ResourceRequirements{}
		}

		if newConfig.Resources.Limits != nil {
			if base.Resources.Limits == nil {
				base.Resources.Limits = make(corev1.ResourceList)
			}

			if newConfig.Resources.Limits.Cpu != nil {
				quantity, err := resource.ParseQuantity(*newConfig.Resources.Limits.Cpu)
				if err != nil {
					return err
				}
				base.Resources.Limits[corev1.ResourceCPU] = quantity
			}
			if newConfig.Resources.Limits.Memory != nil {
				quantity, err := resource.ParseQuantity(*newConfig.Resources.Limits.Memory)
				if err != nil {
					return err
				}
				base.Resources.Limits[corev1.ResourceMemory] = quantity
			}
		}

		if newConfig.Resources.Requests != nil {
			if base.Resources.Requests == nil {
				base.Resources.Requests = make(corev1.ResourceList)
			}

			if newConfig.Resources.Requests.Cpu != nil {
				quantity, err := resource.ParseQuantity(*newConfig.Resources.Requests.Cpu)
				if err != nil {
					return err
				}
				base.Resources.Requests[corev1.ResourceCPU] = quantity
			}
			if newConfig.Resources.Requests.Memory != nil {
				quantity, err := resource.ParseQuantity(*newConfig.Resources.Requests.Memory)
				if err != nil {
					return err
				}
				base.Resources.Requests[corev1.ResourceMemory] = quantity
			}
		}
	}

	if newConfig.HPASpec != nil {
		if base.HpaSpec == nil {
			base.HpaSpec = &autoscalingv2.HorizontalPodAutoscalerSpec{}
		}
		if newConfig.HPASpec.MaxReplicas != nil {
			base.HpaSpec.MaxReplicas = *newConfig.HPASpec.MaxReplicas
		}

		if newConfig.HPASpec.MinReplicas != nil {
			base.HpaSpec.MinReplicas = newConfig.HPASpec.MinReplicas
		}
	}

	if newConfig.Strategy != nil {
		if base.Strategy == nil {
			base.Strategy = &appsv1.DeploymentStrategy{
				RollingUpdate: &appsv1.RollingUpdateDeployment{},
			}
		}
		if newConfig.Strategy.RollingUpdate.MaxSurge != nil {
			switch newConfig.Strategy.RollingUpdate.MaxSurge.Type {
			case intstr.Int:
				base.Strategy.RollingUpdate.MaxSurge = &intstr.IntOrString{
					Type:   0,
					IntVal: newConfig.Strategy.RollingUpdate.MaxSurge.IntVal,
					StrVal: "",
				}
			case intstr.String:
				base.Strategy.RollingUpdate.MaxSurge = &intstr.IntOrString{
					Type:   1,
					IntVal: 0,
					StrVal: newConfig.Strategy.RollingUpdate.MaxSurge.StrVal,
				}
			}
		}

		if newConfig.Strategy.RollingUpdate.MaxUnavailable != nil {
			switch newConfig.Strategy.RollingUpdate.MaxUnavailable.Type {
			case intstr.Int:
				base.Strategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{
					Type:   intstr.Int,
					IntVal: newConfig.Strategy.RollingUpdate.MaxUnavailable.IntVal,
					StrVal: "",
				}
			case intstr.String:
				base.Strategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{
					Type:   intstr.String,
					IntVal: 0,
					StrVal: newConfig.Strategy.RollingUpdate.MaxUnavailable.StrVal,
				}
			}
		}
	}
	return nil
}
