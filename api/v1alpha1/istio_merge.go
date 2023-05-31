package v1alpha1

import (
	"encoding/json"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"istio.io/api/operator/v1alpha1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"google.golang.org/protobuf/types/known/structpb"
	meshv1alpha1 "istio.io/api/mesh/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/pkg/util/protomarshal"
)

const (
	cpu            string = "cpu"
	memory         string = "memory"
	globalField           = "global"
	proxyField            = "proxy"
	resourcesField        = "resources"
	limitsField           = "limits"
	requestsField         = "requests"
)

func (i *Istio) MergeInto(op istioOperator.IstioOperator) (istioOperator.IstioOperator, error) {
	if op.Spec == nil {
		op.Spec = &v1alpha1.IstioOperatorSpec{}
	}

	mergedConfigOp, err := i.mergeConfig(op)
	if err != nil {
		return op, err
	}

	mergedResourcesOp, err := i.mergeResources(mergedConfigOp)
	if err != nil {
		return op, err
	}

	return mergedResourcesOp, nil
}

func (i *Istio) mergeConfig(op istioOperator.IstioOperator) (istioOperator.IstioOperator, error) {
	if i.Spec.Config.NumTrustedProxies == nil {
		return op, nil
	}

	if op.Spec.MeshConfig == nil {
		op.Spec.MeshConfig = &structpb.Struct{}
	}

	c, err := unmarshalMeshConfig(op.Spec.MeshConfig)
	if err != nil {
		return op, err
	}

	if c.DefaultConfig == nil {
		c.DefaultConfig = &meshv1alpha1.ProxyConfig{}
	}

	// Since the gateway topology is not part of the default configuration, and we do not explicitly set it in the
	// chart, it could be nil.
	if c.DefaultConfig.GatewayTopology != nil {
		c.DefaultConfig.GatewayTopology.NumTrustedProxies = uint32(*i.Spec.Config.NumTrustedProxies)
	} else {
		c.DefaultConfig.GatewayTopology = &meshv1alpha1.Topology{NumTrustedProxies: uint32(*i.Spec.Config.NumTrustedProxies)}
	}

	if updatedConfig, err := marshalMeshConfig(c); err != nil {
		return op, err
	} else {
		op.Spec.MeshConfig = updatedConfig
	}

	return op, nil
}

func (i *Istio) mergeResources(op istioOperator.IstioOperator) (istioOperator.IstioOperator, error) {
	if i.Spec.Components == nil {
		return op, nil
	}
	if len(i.Spec.Components.IngressGateways) > 0 {
		if op.Spec.Components == nil {
			op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
		}
		if len(op.Spec.Components.IngressGateways) == 0 {
			op.Spec.Components.IngressGateways = []*v1alpha1.GatewaySpec{}
		}
		for i, gateway := range i.Spec.Components.IngressGateways {
			if len(op.Spec.Components.IngressGateways) <= i {
				op.Spec.Components.IngressGateways = append(op.Spec.Components.IngressGateways, &v1alpha1.GatewaySpec{})
			}
			if op.Spec.Components.IngressGateways[i].K8S == nil {
				op.Spec.Components.IngressGateways[i].K8S = &v1alpha1.KubernetesResourcesSpec{}
			}

			err := mergeK8sConfig(op.Spec.Components.IngressGateways[i].K8S, *gateway.K8s)
			if err != nil {
				return op, err
			}
		}
	}
	if i.Spec.Components.Pilot != nil {
		if op.Spec.Components == nil {
			op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
		}
		if op.Spec.Components.Pilot == nil {
			op.Spec.Components.Pilot = &v1alpha1.ComponentSpec{}
		}
		if op.Spec.Components.Pilot.K8S == nil {
			op.Spec.Components.Pilot.K8S = &v1alpha1.KubernetesResourcesSpec{}
		}
		err := mergeK8sConfig(op.Spec.Components.Pilot.K8S, *i.Spec.Components.Pilot.K8s)
		if err != nil {
			return op, err
		}
	}

	if i.Spec.Components.Proxy != nil && i.Spec.Components.Proxy.K8S.Resources != nil {
		if op.Spec.Values == nil {
			op.Spec.Values = &structpb.Struct{}
			op.Spec.Values.Fields = make(map[string]*structpb.Value)
		}
		global := op.Spec.Values.Fields[globalField].GetStructValue()
		if global == nil {
			global = &structpb.Struct{}
			global.Fields = make(map[string]*structpb.Value)
		}

		proxy := global.Fields[proxyField].GetStructValue()
		if proxy == nil {
			proxy = &structpb.Struct{}
			proxy.Fields = make(map[string]*structpb.Value)
		}

		resources := proxy.Fields[resourcesField].GetStructValue()
		if resources == nil {
			resources = &structpb.Struct{}
			resources.Fields = make(map[string]*structpb.Value)
		}

		if i.Spec.Components.Proxy.K8S.Resources.Limits != nil {
			limits := resources.Fields[limitsField].GetStructValue()
			if limits == nil {
				limits = &structpb.Struct{}
				limits.Fields = make(map[string]*structpb.Value)
			}
			if i.Spec.Components.Proxy.K8S.Resources.Limits.Cpu != nil {
				limits.Fields[cpu] = structpb.NewStringValue(*i.Spec.Components.Proxy.K8S.Resources.Limits.Cpu)
			}

			if i.Spec.Components.Proxy.K8S.Resources.Limits.Memory != nil {
				limits.Fields[memory] = structpb.NewStringValue(*i.Spec.Components.Proxy.K8S.Resources.Limits.Memory)
			}

			l, err := structpb.NewValue(limits.AsMap())
			if err != nil {
				return op, err
			}
			resources.Fields[limitsField] = l
		}

		if i.Spec.Components.Proxy.K8S.Resources.Requests != nil {
			requests := resources.Fields[requestsField].GetStructValue()
			if requests == nil {
				requests = &structpb.Struct{}
				requests.Fields = make(map[string]*structpb.Value)
			}
			if i.Spec.Components.Proxy.K8S.Resources.Requests.Cpu != nil {
				requests.Fields[cpu] = structpb.NewStringValue(*i.Spec.Components.Proxy.K8S.Resources.Requests.Cpu)
			}

			if i.Spec.Components.Proxy.K8S.Resources.Requests.Memory != nil {
				requests.Fields[memory] = structpb.NewStringValue(*i.Spec.Components.Proxy.K8S.Resources.Requests.Memory)
			}

			r, err := structpb.NewValue(requests.AsMap())
			if err != nil {
				return op, err
			}
			resources.Fields[requestsField] = r
		}

		r, err := structpb.NewValue(resources.AsMap())
		if err != nil {
			return op, err
		}
		proxy.Fields[resourcesField] = r

		p, err := structpb.NewValue(proxy.AsMap())
		if err != nil {
			return op, err
		}
		global.Fields[proxyField] = p

		g, err := structpb.NewValue(global.AsMap())
		if err != nil {
			return op, err
		}
		op.Spec.Values.Fields[globalField] = g
	}

	if i.Spec.Components.Cni != nil {
		if op.Spec.Components == nil {
			op.Spec.Components = &v1alpha1.IstioComponentSetSpec{}
		}

		if op.Spec.Components.Cni == nil {
			op.Spec.Components.Cni = &v1alpha1.ComponentSpec{}
		}

		if op.Spec.Components.Cni.K8S == nil {
			op.Spec.Components.Cni.K8S = &v1alpha1.KubernetesResourcesSpec{}
		}

		if op.Spec.Components.Cni.K8S.Affinity == nil {
			op.Spec.Components.Cni.K8S.Affinity = &v1alpha1.Affinity{}
		}

		if i.Spec.Components.Cni.K8S.Affinity != nil {
			if op.Spec.Components.Cni.K8S.Affinity == nil {
				op.Spec.Components.Cni.K8S.Affinity = &v1alpha1.Affinity{}
			}
			if i.Spec.Components.Cni.K8S.Affinity.PodAffinity != nil {
				if op.Spec.Components.Cni.K8S.Affinity.PodAffinity == nil {
					op.Spec.Components.Cni.K8S.Affinity.PodAffinity = &v1alpha1.PodAffinity{}
				}
				op.Spec.Components.Cni.K8S.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []*v1alpha1.WeightedPodAffinityTerm{}
				for _, term := range i.Spec.Components.Cni.K8S.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
					var w v1alpha1.WeightedPodAffinityTerm
					w.Weight = term.Weight

					var v v1alpha1.PodAffinityTerm
					v.TopologyKey = term.PodAffinityTerm.TopologyKey
					v.Namespaces = term.PodAffinityTerm.Namespaces
					v.LabelSelector = term.PodAffinityTerm.LabelSelector

					w.PodAffinityTerm = &v

					op.Spec.Components.Cni.K8S.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(op.Spec.Components.Cni.K8S.Affinity.PodAffinity.PreferredDuringSchedulingIgnoredDuringExecution, &w)
				}

				op.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []*v1alpha1.PodAffinityTerm{}
				for _, term := range i.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
					var v v1alpha1.PodAffinityTerm
					v.TopologyKey = term.TopologyKey
					v.Namespaces = term.Namespaces
					v.LabelSelector = term.LabelSelector

					op.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(op.Spec.Components.Cni.K8S.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution, &v)
				}
			}

			if i.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity != nil {
				if op.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity == nil {
					op.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity = &v1alpha1.PodAntiAffinity{}
				}
				op.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []*v1alpha1.WeightedPodAffinityTerm{}
				for _, term := range i.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
					var w v1alpha1.WeightedPodAffinityTerm
					w.Weight = term.Weight

					var v v1alpha1.PodAffinityTerm
					v.TopologyKey = term.PodAffinityTerm.TopologyKey
					v.Namespaces = term.PodAffinityTerm.Namespaces
					v.LabelSelector = term.PodAffinityTerm.LabelSelector

					w.PodAffinityTerm = &v

					op.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(op.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution, &w)
				}

				op.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = []*v1alpha1.PodAffinityTerm{}
				for _, term := range i.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution {
					var v v1alpha1.PodAffinityTerm
					v.TopologyKey = term.TopologyKey
					v.Namespaces = term.Namespaces
					v.LabelSelector = term.LabelSelector

					op.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(op.Spec.Components.Cni.K8S.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution, &v)
				}
			}

			if i.Spec.Components.Cni.K8S.Affinity.NodeAffinity != nil {
				if op.Spec.Components.Cni.K8S.Affinity.NodeAffinity == nil {
					op.Spec.Components.Cni.K8S.Affinity.NodeAffinity = &v1alpha1.NodeAffinity{}
				}
				op.Spec.Components.Cni.K8S.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = []*v1alpha1.PreferredSchedulingTerm{}
				for _, term := range i.Spec.Components.Cni.K8S.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution {
					var w v1alpha1.PreferredSchedulingTerm
					w.Weight = term.Weight

					var v v1alpha1.NodeSelectorTerm
					v.MatchExpressions = []*v1alpha1.NodeSelectorRequirement{}
					for _, expression := range term.Preference.MatchExpressions {
						n := v1alpha1.NodeSelectorRequirement{
							Key:      expression.Key,
							Operator: string(expression.Operator),
							Values:   expression.Values,
						}
						v.MatchExpressions = append(v.MatchExpressions, &n)
					}

					v.MatchFields = []*v1alpha1.NodeSelectorRequirement{}
					for _, expression := range term.Preference.MatchFields {
						n := v1alpha1.NodeSelectorRequirement{
							Key:      expression.Key,
							Operator: string(expression.Operator),
							Values:   expression.Values,
						}
						v.MatchFields = append(v.MatchFields, &n)
					}

					w.Preference = &v

					op.Spec.Components.Cni.K8S.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(op.Spec.Components.Cni.K8S.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution, &w)
				}

				op.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &v1alpha1.NodeSelector{}
				if i.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
					for _, term := range i.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
						var t v1alpha1.NodeSelectorTerm
						t.MatchExpressions = []*v1alpha1.NodeSelectorRequirement{}
						for _, expression := range term.MatchExpressions {
							n := v1alpha1.NodeSelectorRequirement{
								Key:      expression.Key,
								Operator: string(expression.Operator),
								Values:   expression.Values,
							}
							t.MatchExpressions = append(t.MatchExpressions, &n)
						}

						t.MatchFields = []*v1alpha1.NodeSelectorRequirement{}
						for _, expression := range term.MatchFields {
							n := v1alpha1.NodeSelectorRequirement{
								Key:      expression.Key,
								Operator: string(expression.Operator),
								Values:   expression.Values,
							}
							t.MatchFields = append(t.MatchFields, &n)
						}

						op.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms =
							append(op.Spec.Components.Cni.K8S.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, &t)
					}
				}
			}
		}

		if i.Spec.Components.Cni.K8S.Resources != nil {
			if op.Spec.Components.Cni.K8S.Resources == nil {
				op.Spec.Components.Cni.K8S.Resources = &v1alpha1.Resources{}
			}

			if i.Spec.Components.Cni.K8S.Resources.Limits != nil {
				if op.Spec.Components.Cni.K8S.Resources.Limits == nil {
					op.Spec.Components.Cni.K8S.Resources.Limits = map[string]string{}
				}
				if i.Spec.Components.Cni.K8S.Resources.Limits.Cpu != nil {
					op.Spec.Components.Cni.K8S.Resources.Limits[cpu] = *i.Spec.Components.Cni.K8S.Resources.Limits.Cpu
				}
				if i.Spec.Components.Cni.K8S.Resources.Limits.Memory != nil {
					op.Spec.Components.Cni.K8S.Resources.Limits[memory] = *i.Spec.Components.Cni.K8S.Resources.Limits.Memory
				}
			}

			if i.Spec.Components.Cni.K8S.Resources.Requests != nil {
				if op.Spec.Components.Cni.K8S.Resources.Requests == nil {
					op.Spec.Components.Cni.K8S.Resources.Requests = map[string]string{}
				}
				if i.Spec.Components.Cni.K8S.Resources.Requests.Cpu != nil {
					op.Spec.Components.Cni.K8S.Resources.Requests[cpu] = *i.Spec.Components.Cni.K8S.Resources.Requests.Cpu
				}
				if i.Spec.Components.Cni.K8S.Resources.Requests.Memory != nil {
					op.Spec.Components.Cni.K8S.Resources.Requests[memory] = *i.Spec.Components.Cni.K8S.Resources.Requests.Memory
				}
			}
		}
	}

	return op, nil
}

func mergeK8sConfig(base *v1alpha1.KubernetesResourcesSpec, newConfig KubernetesResourcesConfig) error {
	if newConfig.Resources != nil {
		if base.Resources == nil {
			base.Resources = &v1alpha1.Resources{}
		}

		if newConfig.Resources.Limits != nil {
			if base.Resources.Limits == nil {
				base.Resources.Limits = map[string]string{}
			}
			if newConfig.Resources.Limits.Cpu != nil {
				base.Resources.Limits[cpu] = *newConfig.Resources.Limits.Cpu
			}
			if newConfig.Resources.Limits.Memory != nil {
				base.Resources.Limits[memory] = *newConfig.Resources.Limits.Memory
			}
		}

		if newConfig.Resources.Requests != nil {
			if base.Resources.Requests == nil {
				base.Resources.Requests = map[string]string{}
			}
			if newConfig.Resources.Requests.Cpu != nil {
				base.Resources.Requests[cpu] = *newConfig.Resources.Requests.Cpu
			}
			if newConfig.Resources.Requests.Memory != nil {
				base.Resources.Requests[memory] = *newConfig.Resources.Requests.Memory
			}
		}
	}

	if newConfig.HPASpec != nil {
		if base.HpaSpec == nil {
			base.HpaSpec = &v1alpha1.HorizontalPodAutoscalerSpec{}
		}
		if newConfig.HPASpec.MaxReplicas != nil {
			base.HpaSpec.MaxReplicas = *newConfig.HPASpec.MaxReplicas
		}

		if newConfig.HPASpec.MinReplicas != nil {
			base.HpaSpec.MinReplicas = *newConfig.HPASpec.MinReplicas
		}
	}

	if newConfig.Strategy != nil {
		if base.Strategy == nil {
			base.Strategy = &v1alpha1.DeploymentStrategy{
				RollingUpdate: &v1alpha1.RollingUpdateDeployment{},
			}
		}
		if newConfig.Strategy.RollingUpdate.MaxSurge != nil {
			switch newConfig.Strategy.RollingUpdate.MaxSurge.Type {
			case intstr.Int:
				base.Strategy.RollingUpdate.MaxSurge = &v1alpha1.IntOrString{
					Type:   0,
					IntVal: wrapperspb.Int32(int32(newConfig.Strategy.RollingUpdate.MaxSurge.IntVal)),
					StrVal: nil,
				}
			case intstr.String:
				base.Strategy.RollingUpdate.MaxSurge = &v1alpha1.IntOrString{
					Type:   1,
					IntVal: nil,
					StrVal: wrapperspb.String(newConfig.Strategy.RollingUpdate.MaxSurge.StrVal),
				}
			}
		}

		if newConfig.Strategy.RollingUpdate.MaxUnavailable != nil {
			switch newConfig.Strategy.RollingUpdate.MaxUnavailable.Type {
			case intstr.Int:
				base.Strategy.RollingUpdate.MaxUnavailable = &v1alpha1.IntOrString{
					Type:   int64(intstr.Int),
					IntVal: wrapperspb.Int32(int32(newConfig.Strategy.RollingUpdate.MaxUnavailable.IntVal)),
					StrVal: nil,
				}
			case intstr.String:
				base.Strategy.RollingUpdate.MaxUnavailable = &v1alpha1.IntOrString{
					Type:   int64(intstr.String),
					IntVal: nil,
					StrVal: wrapperspb.String(newConfig.Strategy.RollingUpdate.MaxUnavailable.StrVal),
				}
			}
		}
	}
	return nil
}

func unmarshalMeshConfig(s *structpb.Struct) (*meshv1alpha1.MeshConfig, error) {
	var c meshv1alpha1.MeshConfig

	jsonStr, err := json.Marshal(s.Fields)
	if err != nil {
		return &c, err
	}

	if err := protomarshal.Unmarshal(jsonStr, &c); err != nil {
		return &c, err
	}

	return &c, nil
}

func marshalMeshConfig(m *meshv1alpha1.MeshConfig) (*structpb.Struct, error) {
	mMap, err := protomarshal.ToJSONMap(m)
	if err != nil {
		return nil, err
	}

	return structpb.NewStruct(mMap)
}
