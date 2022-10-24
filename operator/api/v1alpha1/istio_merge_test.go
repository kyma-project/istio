package v1alpha1_test

import (
	"testing"

	istioOperatorSpec "istio.io/api/operator/v1alpha1"

	"github.com/brianvoe/gofakeit"
	"github.com/kyma-project/istio/operator/api/v1alpha1"
	"github.com/kyma-project/istio/operator/pkg/lib/builder"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	istioOperator "istio.io/istio/operator/pkg/apis/istio/v1alpha1"
)

func Test_MergeWith(t *testing.T) {
	// Given
	someKymaOperator := v1alpha1.Istio{}
	gofakeit.Struct(&someKymaOperator)
	someKymaOperator.Spec.Controlplane.MeshConfig.GatewayTopology.NumTrustedProxies = 4

	someIstioOperator := istioOperator.IstioOperator{
		Spec: &istioOperatorSpec.IstioOperatorSpec{
			MeshConfig: &structpb.Struct{},
		},
	}
	gatewayTopo, err := structpb.NewValue(map[string]interface{}{
		"numTrustedProxies": 5,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(someIstioOperator.Spec.MeshConfig.Fields) == 0 {
		someIstioOperator.Spec.MeshConfig.Fields = make(map[string]*structpb.Value)
	}

	someIstioOperator.Spec.MeshConfig.Fields["gatewayTopology"] = gatewayTopo

	// When
	operatorBuilder := builder.NewIstioOperatorBuilder(someIstioOperator)
	out, err := operatorBuilder.MergeWith(someKymaOperator).Get()
	_, errString := operatorBuilder.MergeWith(someKymaOperator).GetString()
	_, errJson := operatorBuilder.MergeWith(someKymaOperator).GetJSONByteArray()

	// Then
	require.NoError(t, err)
	require.Equal(t, someKymaOperator.Spec.Controlplane.MeshConfig.GatewayTopology.NumTrustedProxies,
		int(out.Spec.MeshConfig.Fields["gatewayTopology"].
			GetStructValue().Fields["numTrustedProxies"].GetNumberValue()))

	require.NoError(t, errString)
	require.NoError(t, errJson)
}

func Test_NoBaseOperator(t *testing.T) {
	// Given
	someKymaOperator := v1alpha1.Istio{}
	gofakeit.Struct(&someKymaOperator)
	someKymaOperator.Spec.Controlplane.MeshConfig.GatewayTopology.NumTrustedProxies = 4

	// When
	operatorBuilder := builder.NewIstioOperatorBuilder()
	out, err := operatorBuilder.MergeWith(someKymaOperator).Get()

	// Then
	require.NoError(t, err)
	require.Equal(t, someKymaOperator.Spec.Controlplane.MeshConfig.GatewayTopology.NumTrustedProxies,
		int(out.Spec.MeshConfig.Fields["gatewayTopology"].
			GetStructValue().Fields["numTrustedProxies"].GetNumberValue()))
}
