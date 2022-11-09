package v1alpha1

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"istio.io/api/mesh/v1alpha1"
	operatorv1alpha1 "istio.io/api/operator/v1alpha1"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
	"istio.io/istio/pkg/config/mesh"
	"istio.io/istio/pkg/util/protomarshal"
)

func TestIstioMergeInto(t *testing.T) {

	t.Run("Should update numTrustedProxies on IstioOperator from 1 to 5", func(t *testing.T) {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &v1alpha1.Topology{NumTrustedProxies: 1}
		meshConfig, err := convert(m)
		if err != nil {
			t.Error(err)
		}

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
		require.NoError(t, err)
		require.Equal(t, float64(5), out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue())

	})

	t.Run("Should set numTrustedProxies on IstioOperator to 5 when no GatewayTopology is configured", func(t *testing.T) {
		// given
		m := mesh.DefaultMeshConfig()
		meshConfig, err := convert(m)
		if err != nil {
			t.Error(err)
		}

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
		require.NoError(t, err)
		require.Equal(t, float64(numProxies), out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue())

	})

	t.Run("Should change nothing if config is empty", func(t *testing.T) {
		// given
		m := mesh.DefaultMeshConfig()
		m.DefaultConfig.GatewayTopology = &v1alpha1.Topology{NumTrustedProxies: 1}
		meshConfig, err := convert(m)
		if err != nil {
			t.Error(err)
		}

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}

		istioCR := Istio{Spec: IstioSpec{}}

		// when
		out, err := istioCR.MergeInto(iop)

		// then
		require.NoError(t, err)
		require.Equal(t, float64(1), out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue())

	})
	t.Run("Should set numTrustedProxies on IstioOperator to 5 when there is no defaultConfig in meshConfig", func(t *testing.T) {
		// given
		m := &v1alpha1.MeshConfig{
			EnableTracing: true,
		}
		meshConfig, err := convert(m)
		if err != nil {
			t.Error(err)
		}

		iop := istioOperator.IstioOperator{
			Spec: &operatorv1alpha1.IstioOperatorSpec{
				MeshConfig: meshConfig,
			},
		}
		numProxies := 5

		istioCR := Istio{Spec: IstioSpec{Config: Config{NumTrustedProxies: &numProxies}}}

		// when
		out, err := istioCR.MergeInto(iop)
		fmt.Println(out)
		fmt.Println(iop)

		// then
		require.NoError(t, err)
		require.Equal(t, float64(5), out.Spec.MeshConfig.Fields["defaultConfig"].
			GetStructValue().Fields["gatewayTopology"].GetStructValue().Fields["numTrustedProxies"].GetNumberValue())
	})
}

func convert(a *v1alpha1.MeshConfig) (*structpb.Struct, error) {

	mMap, err := protomarshal.ToJSONMap(a)
	if err != nil {
		return nil, err
	}

	if mStruct, err := structpb.NewStruct(mMap); err != nil {
		return nil, err
	} else {
		return mStruct, nil
	}
}
