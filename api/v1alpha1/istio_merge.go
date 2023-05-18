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
	cpu    string = "cpu"
	memory string = "memory"
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

			err := mergeK8sConfig(op.Spec.Components.IngressGateways[i].K8S, gateway.K8s)
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
		err := mergeK8sConfig(op.Spec.Components.Pilot.K8S, i.Spec.Components.Pilot.K8s)
		if err != nil {
			return op, err
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
			if newConfig.Resources.Requests != nil {
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
