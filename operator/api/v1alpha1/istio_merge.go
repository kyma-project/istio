package v1alpha1

import (
	"encoding/json"
	"google.golang.org/protobuf/types/known/structpb"
	meshv1alpha1 "istio.io/api/mesh/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/pkg/util/protomarshal"
)

func (i *Istio) MergeInto(op istioOperator.IstioOperator) (istioOperator.IstioOperator, error) {

	c, err := unmarshalMeshConfig(op.Spec.MeshConfig)
	if err != nil {
		return op, err
	}

	// Since the gateway topology is not part of the default configuration, and we do not explicitly set it in the
	// chart, it could be nil.
	if c.DefaultConfig.GatewayTopology != nil {
		c.DefaultConfig.GatewayTopology.NumTrustedProxies = uint32(i.Spec.Config.NumTrustedProxies)
	} else {
		c.DefaultConfig.GatewayTopology = &meshv1alpha1.Topology{NumTrustedProxies: uint32(i.Spec.Config.NumTrustedProxies)}
	}

	if updatedConfig, err := marshalMeshConfig(c); err != nil {
		return op, err
	} else {
		op.Spec.MeshConfig = updatedConfig
	}

	return op, nil
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
